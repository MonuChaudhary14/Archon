package auth

import (
	"context"
	"crypto/subtle"
	"time"
	"unicode"
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

func (s *authService) Register(
	ctx context.Context,
	req RegisterRequest,
) (*UserResponse, error) {

	existingUser, err :=
		s.userRepository.FindByEmail(
			ctx,
			req.Email,
		)

	if err == nil && existingUser != nil {
		return nil, ErrEmailExists
	}

	if s.otpStore.CheckOTPCooldown(ctx, req.Email) {
		return nil, ErrPleaseWait
	}

	if !isStrongPassword(req.Password) {
		return nil, ErrWeakPassword
	}

	hashedPassword, err :=
		HashPassword(req.Password)

	if err != nil {
		return nil, err
	}

	unverifiedUser := &UnverifiedUser{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hashedPassword,
	}

	otp, err := GenerateOTP()

	if err != nil {
		return nil, err
	}

	otpHash := HashOTP(
		otp,
	)

	err = s.otpStore.SaveVerificationOTP(
		ctx,
		req.Email,
		otpHash,
	)

	if err != nil {
		return nil, err
	}

	err = s.otpStore.SaveUnverifiedUser(
		ctx,
		req.Email,
		unverifiedUser,
	)

	if err != nil {
		return nil, err
	}

	_ = s.otpStore.SetOTPCooldown(ctx, req.Email)

	err = s.emailQueue.Enqueue(

		ctx,
		EmailJob{
			Email: req.Email,
			OTP:   otp,
		},
	)

	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:    0,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func (s *authService) Login(
	ctx context.Context,
	req LoginRequest,
) (*LoginResponse, error) {

	user, err := s.userRepository.FindByEmail(
		ctx,
		req.Email,
	)

	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsVerified {
		return nil, ErrEmailNotVerified
	}

	err = CheckPassword(
		req.Password,
		user.PasswordHash,
	)

	if err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := GenerateAccessToken(
		user,
		s.jwtSecret,
	)

	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken()

	if err != nil {
		return nil, err
	}

	refreshTokenHash := HashToken(
		refreshToken,
	)

	err = s.userRepository.SaveRefreshToken(
		ctx,
		&RefreshToken{
			UserID:    user.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().Add(
				30 * 24 * time.Hour,
			),
		},
	)

	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) VerifyEmail(
	ctx context.Context,
	req VerifyEmailRequest,
) error {

	storedHash, err :=
		s.otpStore.GetVerificationOTP(
			ctx,
			req.Email,
		)

	if err != nil {
		return ErrInvalidOTP
	}

	incomingHash :=
		HashOTP(req.OTP)

	if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(storedHash)) != 1 {
		return ErrInvalidOTP
	}

	unverifiedUser, err := s.otpStore.GetUnverifiedUser(ctx, req.Email)
	if err != nil {
		return ErrInvalidOTP
	}

	user := &User{
		Name:         unverifiedUser.Name,
		Email:        unverifiedUser.Email,
		PasswordHash: unverifiedUser.PasswordHash,
		IsVerified:   true,
	}

	err = s.userRepository.CreateUser(
		ctx,
		user,
	)

	if err != nil {
		return err
	}

	_ = s.otpStore.DeleteVerificationOTP(
		ctx,
		req.Email,
	)

	_ = s.otpStore.DeleteUnverifiedUser(
		ctx,
		req.Email,
	)

	return nil
}

func (s *authService) Refresh(
	ctx context.Context,
	req RefreshRequest,
) (*LoginResponse, error) {

	var accessToken, newRefreshToken string

	err := s.userRepository.RunInTx(ctx, func(txRepo UserRepository) error {
		hashedIncoming := HashToken(req.RefreshToken)
		storedToken, err := txRepo.FindRefreshTokenByHash(ctx, hashedIncoming)
		if err != nil {
			return ErrInvalidToken
		}

		if storedToken.IsUsed {
			_ = txRepo.RevokeAllUserTokens(ctx, storedToken.UserID)
			return ErrInvalidToken
		}

		if time.Now().After(storedToken.ExpiresAt) {
			_ = txRepo.DeleteRefreshToken(ctx, storedToken.ID)
			return ErrInvalidToken
		}

		_ = txRepo.MarkRefreshTokenAsUsed(ctx, storedToken.ID)

		user, err := txRepo.FindByID(ctx, storedToken.UserID)
		if err != nil {
			return err
		}

		accessToken, err = GenerateAccessToken(user, s.jwtSecret)
		if err != nil {
			return err
		}

		newRefreshToken, err = GenerateRefreshToken()
		if err != nil {
			return err
		}

		newRefreshTokenHash := HashToken(newRefreshToken)

		return txRepo.SaveRefreshToken(
			ctx,
			&RefreshToken{
				UserID:    user.ID,
				TokenHash: newRefreshTokenHash,
				ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			},
		)
	})

	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *authService) Logout(
	ctx context.Context,
	req LogoutRequest,
) error {
	hashedIncoming := HashToken(req.RefreshToken)
	storedToken, err := s.userRepository.FindRefreshTokenByHash(ctx, hashedIncoming)
	if err != nil {
		return ErrInvalidToken
	}

	_ = s.userRepository.DeleteRefreshToken(ctx, storedToken.ID)
	return nil
}

func (s *authService) ForgotPassword(
	ctx context.Context,
	req ForgotPasswordRequest,
) error {
	user, err := s.userRepository.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	otp, err := GenerateOTP()
	if err != nil {
		return err
	}

	otpHash := HashOTP(otp)
	err = s.otpStore.SavePasswordResetOTP(ctx, user.Email, otpHash)
	if err != nil {
		return err
	}

	err = s.emailQueue.Enqueue(
		ctx,
		EmailJob{
			Email: user.Email,
			OTP:   otp,
		},
	)

	return err
}

func (s *authService) VerifyResetOTP(
	ctx context.Context,
	req VerifyResetOTPRequest,
) (*VerifyResetOTPResponse, error) {
	storedHash, err := s.otpStore.GetPasswordResetOTP(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidOTP
	}

	incomingHash := HashOTP(req.OTP)
	if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(storedHash)) != 1 {
		return nil, ErrInvalidOTP
	}

	// Valid OTP. Delete it so it can't be reused.
	_ = s.otpStore.DeletePasswordResetOTP(ctx, req.Email)

	token, err := GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	if err := s.otpStore.SaveResetToken(ctx, token, req.Email); err != nil {
		return nil, err
	}

	return &VerifyResetOTPResponse{
		ResetToken: token,
	}, nil
}

func (s *authService) ResetPassword(
	ctx context.Context,
	req ResetPasswordRequest,
) error {
	email, err := s.otpStore.GetEmailByResetToken(ctx, req.ResetToken)
	if err != nil {
		return ErrInvalidToken
	}

	user, err := s.userRepository.FindByEmail(ctx, email)
	if err != nil {
		return ErrInvalidToken
	}

	if !isStrongPassword(req.NewPassword) {
		return ErrWeakPassword
	}

	hashedPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	err = s.userRepository.RunInTx(ctx, func(txRepo UserRepository) error {
		if err := txRepo.UpdatePassword(ctx, user.Email, hashedPassword); err != nil {
			return err
		}
		// Revoke all tokens within the transaction
		_ = txRepo.RevokeAllUserTokens(ctx, user.ID)
		return nil
	})

	if err == nil {
		// Only delete the reset token if the transaction was completely successful
		_ = s.otpStore.DeleteResetToken(ctx, req.ResetToken)
	}

	return err
}

func (s *authService) ResendOTP(
	ctx context.Context,
	req ResendOTPRequest,
) error {
	if s.otpStore.CheckOTPCooldown(ctx, req.Email) {
		return ErrPleaseWait
	}

	if req.Intent == "register" {
		unverifiedUser, err := s.otpStore.GetUnverifiedUser(ctx, req.Email)
		if err != nil {
			return ErrSessionExpired
		}

		otp, err := GenerateOTP()
		if err != nil {
			return err
		}

		otpHash := HashOTP(otp)
		if err := s.otpStore.SaveVerificationOTP(ctx, req.Email, otpHash); err != nil {
			return err
		}

		// Refresh the UnverifiedUser TTL
		if err := s.otpStore.SaveUnverifiedUser(ctx, req.Email, unverifiedUser); err != nil {
			return err
		}

		_ = s.otpStore.SetOTPCooldown(ctx, req.Email)

		return s.emailQueue.Enqueue(ctx, EmailJob{
			Email: req.Email,
			OTP:   otp,
		})
	} else if req.Intent == "forgot_password" {
		user, err := s.userRepository.FindByEmail(ctx, req.Email)
		if err != nil {
			return nil // Silently ignore if user not found for security
		}

		otp, err := GenerateOTP()
		if err != nil {
			return err
		}

		otpHash := HashOTP(otp)
		if err := s.otpStore.SavePasswordResetOTP(ctx, user.Email, otpHash); err != nil {
			return err
		}

		_ = s.otpStore.SetOTPCooldown(ctx, req.Email)

		return s.emailQueue.Enqueue(ctx, EmailJob{
			Email: user.Email,
			OTP:   otp,
		})
	}

	return nil
}

func isStrongPassword(s string) bool {
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range s {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial && len(s) >= 8 && len(s) <= 30
}
