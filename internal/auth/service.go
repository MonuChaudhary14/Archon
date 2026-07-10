package auth

import (
	"context"
)

type AuthService interface {
	Register(
		ctx context.Context,
		req RegisterRequest,
	) (*UserResponse, error)

	Login(
		ctx context.Context,
		req LoginRequest,
	) (*LoginResponse, error)

	OAuthLogin(
		ctx context.Context,
		provider string,
		userInfo *OAuthUserInfo,
	) (*LoginResponse, error)

	VerifyEmail(
		ctx context.Context,
		req VerifyEmailRequest,
	) error

	Refresh(
		ctx context.Context,
		req RefreshRequest,
	) (*LoginResponse, error)

	Logout(
		ctx context.Context,
		req LogoutRequest,
	) error

	ForgotPassword(
		ctx context.Context,
		req ForgotPasswordRequest,
	) error

	ResetPassword(
		ctx context.Context,
		req ResetPasswordRequest,
	) error

	VerifyResetOTP(
		ctx context.Context,
		req VerifyResetOTPRequest,
	) (*VerifyResetOTPResponse, error)

	ResendOTP(
		ctx context.Context,
		req ResendOTPRequest,
	) error
}

type authService struct {
	userRepository UserRepository
	otpStore       *OTPStore
	mailService    *MailService
	emailQueue     *EmailQueue
	jwtSecret      string
}

func NewAuthService(
	userRepository UserRepository,
	otpStore *OTPStore,
	mailService *MailService,
	emailQueue *EmailQueue,
	jwtSecret string,
) AuthService {

	return &authService{
		userRepository: userRepository,
		otpStore:       otpStore,
		mailService:    mailService,
		emailQueue:     emailQueue,
		jwtSecret:      jwtSecret,
	}
}


