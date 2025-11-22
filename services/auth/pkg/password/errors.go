package password

import "errors"

var (
	ErrPasswordTooShort        = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong         = errors.New("password must not exceed 72 characters")
	ErrPasswordNeedsUppercase  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNeedsLowercase  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNeedsNumber     = errors.New("password must contain at least one number")
)
