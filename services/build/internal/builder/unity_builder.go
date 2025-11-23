package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yourorg/collab/services/build/internal/models"
)

// UnityBuilder handles Unity build execution
type UnityBuilder struct {
	unityPath     string
	workspaceDir  string
	logWriter     io.Writer
}

// NewUnityBuilder creates a new Unity builder
func NewUnityBuilder(unityPath, workspaceDir string, logWriter io.Writer) *UnityBuilder {
	return &UnityBuilder{
		unityPath:    unityPath,
		workspaceDir: workspaceDir,
		logWriter:    logWriter,
	}
}

// BuildOptions contains options for Unity build
type BuildOptions struct {
	ProjectPath  string
	BuildTarget  string
	OutputPath   string
	UnityVersion string
	BuildOptions []string
}

// Execute runs the Unity build process
func (b *UnityBuilder) Execute(ctx context.Context, build *models.Build, opts *BuildOptions) error {
	// Create workspace directory for this build
	buildDir := filepath.Join(b.workspaceDir, build.ID)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Clone project repository
	if err := b.cloneRepository(ctx, build, buildDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Prepare Unity command
	args := b.prepareUnityCommand(build, opts, buildDir)

	// Execute Unity build
	cmd := exec.CommandContext(ctx, b.unityPath, args...)
	cmd.Dir = buildDir
	cmd.Stdout = b.logWriter
	cmd.Stderr = b.logWriter

	b.log(fmt.Sprintf("Starting Unity build for project: %s\n", build.ProjectID))
	b.log(fmt.Sprintf("Build target: %s\n", build.BuildTarget))
	b.log(fmt.Sprintf("Unity version: %s\n", build.UnityVersion))
	b.log(fmt.Sprintf("Command: %s %v\n", b.unityPath, args))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unity build failed: %w", err)
	}

	b.log("Build completed successfully\n")

	return nil
}

// prepareUnityCommand prepares Unity CLI arguments
func (b *UnityBuilder) prepareUnityCommand(build *models.Build, opts *BuildOptions, buildDir string) []string {
	args := []string{
		"-quit",
		"-batchmode",
		"-nographics",
		"-projectPath", filepath.Join(buildDir, opts.ProjectPath),
		"-buildTarget", build.BuildTarget,
		"-executeMethod", "BuildScript.Build",
	}

	// Add custom build options
	if len(opts.BuildOptions) > 0 {
		args = append(args, opts.BuildOptions...)
	}

	return args
}

// cloneRepository clones the project git repository
func (b *UnityBuilder) cloneRepository(ctx context.Context, build *models.Build, targetDir string) error {
	b.log(fmt.Sprintf("Cloning repository for commit: %s\n", build.CommitSHA))

	// TODO: Integrate with actual git service
	// For now, this is a placeholder that simulates repository cloning

	// In production, this would:
	// 1. Clone from git repository URL (from project metadata)
	// 2. Checkout specific commit SHA
	// 3. Initialize submodules if needed

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", ".", targetDir)
	cmd.Stdout = b.logWriter
	cmd.Stderr = b.logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Checkout specific commit
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", build.CommitSHA)
	checkoutCmd.Dir = targetDir
	checkoutCmd.Stdout = b.logWriter
	checkoutCmd.Stderr = b.logWriter

	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	b.log("Repository cloned successfully\n")
	return nil
}

// log writes a log message
func (b *UnityBuilder) log(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(b.logWriter, "[%s] %s", timestamp, message)
}

// Cleanup removes build artifacts and temporary files
func (b *UnityBuilder) Cleanup(buildID string) error {
	buildDir := filepath.Join(b.workspaceDir, buildID)
	return os.RemoveAll(buildDir)
}

// GetBuildArtifacts collects build artifacts from output directory
func (b *UnityBuilder) GetBuildArtifacts(buildID, outputPath string) ([]string, error) {
	artifacts := []string{}

	buildDir := filepath.Join(b.workspaceDir, buildID, outputPath)

	err := filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			artifacts = append(artifacts, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	}

	return artifacts, nil
}
