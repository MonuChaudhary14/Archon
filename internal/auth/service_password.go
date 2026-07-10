package auth

import (
	"context"
	"crypto/subtle"

	"github.com/MonuChaudhary14/Archon/pkg/validator"
)

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

	if !validator.IsStrongPassword(req.NewPassword) {
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
		// Increment Token Version to instantly revoke active access tokens
		_ = txRepo.IncrementTokenVersion(ctx, user.ID)
		return nil
	})

	if err == nil {
		// Only delete the reset token if the transaction was completely successful
		_ = s.otpStore.DeleteResetToken(ctx, req.ResetToken)
	}

	return err
}
