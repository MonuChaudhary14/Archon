package auth

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidLogin       = errors.New("invalid email or password")
	ErrInvalidOTP         = errors.New("invalid otp")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrWeakPassword       = errors.New("password must contain uppercase, lowercase, number, and special character")
)
