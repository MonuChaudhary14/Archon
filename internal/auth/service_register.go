package auth

import (
	"context"
	"crypto/subtle"

	"github.com/MonuChaudhary14/Archon/pkg/validator"
)

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

	if !validator.IsStrongPassword(req.Password) {
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

func (s *authService) ResendOTP(
	ctx context.Context,
	req ResendOTPRequest,
) error {
	if s.otpStore.CheckOTPCooldown(ctx, req.Email) {
		return ErrPleaseWait
	}

	switch req.Intent {
	case "register":
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
	case "forgot_password":
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
