package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	config    *oauth2.Config
	name      string
	userInfoURL string
}

// OAuthProviders holds all configured OAuth providers
type OAuthProviders struct {
	providers map[string]*OAuthProvider
}

// UserInfo represents user information from OAuth provider
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar_url"`
	Provider string `json:"provider"`
}

// NewOAuthProviders initializes OAuth providers
func NewOAuthProviders(googleClientID, googleClientSecret, githubClientID, githubClientSecret, redirectURL string) *OAuthProviders {
	providers := make(map[string]*OAuthProvider)

	// Google OAuth
	if googleClientID != "" && googleClientSecret != "" {
		providers["google"] = &OAuthProvider{
			name: "google",
			config: &oauth2.Config{
				ClientID:     googleClientID,
				ClientSecret: googleClientSecret,
				RedirectURL:  redirectURL + "/google/callback",
				Scopes: []string{
					"https://www.googleapis.com/auth/userinfo.email",
					"https://www.googleapis.com/auth/userinfo.profile",
				},
				Endpoint: google.Endpoint,
			},
			userInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		}
	}

	// GitHub OAuth
	if githubClientID != "" && githubClientSecret != "" {
		providers["github"] = &OAuthProvider{
			name: "github",
			config: &oauth2.Config{
				ClientID:     githubClientID,
				ClientSecret: githubClientSecret,
				RedirectURL:  redirectURL + "/github/callback",
				Scopes: []string{
					"user:email",
					"read:user",
				},
				Endpoint: github.Endpoint,
			},
			userInfoURL: "https://api.github.com/user",
		}
	}

	return &OAuthProviders{
		providers: providers,
	}
}

// GetProvider returns an OAuth provider by name
func (p *OAuthProviders) GetProvider(name string) (*OAuthProvider, error) {
	provider, exists := p.providers[name]
	if !exists {
		return nil, fmt.Errorf("OAuth provider '%s' not configured", name)
	}
	return provider, nil
}

// GetAuthURL generates the OAuth authorization URL
func (p *OAuthProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for a token
func (p *OAuthProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return p.config.Exchange(ctx, code)
}

// GetUserInfo fetches user information from the OAuth provider
func (p *OAuthProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var rawInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Parse user info based on provider
	userInfo := &UserInfo{
		Provider: p.name,
	}

	switch p.name {
	case "google":
		userInfo.ID = getString(rawInfo, "id")
		userInfo.Email = getString(rawInfo, "email")
		userInfo.Name = getString(rawInfo, "name")
		userInfo.Avatar = getString(rawInfo, "picture")

	case "github":
		userInfo.ID = fmt.Sprintf("%v", rawInfo["id"])
		userInfo.Name = getString(rawInfo, "name")
		userInfo.Avatar = getString(rawInfo, "avatar_url")

		// GitHub email needs separate API call if not public
		if email := getString(rawInfo, "email"); email != "" {
			userInfo.Email = email
		} else {
			// Fetch email from emails endpoint
			email, err := p.getGitHubEmail(ctx, token)
			if err == nil {
				userInfo.Email = email
			}
		}

	default:
		return nil, fmt.Errorf("unknown provider: %s", p.name)
	}

	return userInfo, nil
}

// getGitHubEmail fetches the primary email from GitHub
func (p *OAuthProvider) getGitHubEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// Fallback to first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// ValidateState validates the OAuth state parameter
func ValidateState(state, expected string) bool {
	return state == expected
}
