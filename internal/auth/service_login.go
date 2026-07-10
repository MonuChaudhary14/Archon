package auth

import (
	"context"
	"time"
)

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

func (s *authService) OAuthLogin(
	ctx context.Context,
	provider string,
	userInfo *OAuthUserInfo,
) (*LoginResponse, error) {

	if userInfo.Email == "" {
		return nil, ErrInvalidCredentials
	}

	connection, err := s.userRepository.FindOAuthConnection(ctx, provider, userInfo.ID)
	var user *User

	if err == nil && connection != nil {
		user, err = s.userRepository.FindByID(ctx, connection.UserID)
		if err != nil {
			return nil, ErrUserNotFound
		}
	} else {
		user, err = s.userRepository.FindByEmail(ctx, userInfo.Email)
		if err != nil {
			user = &User{
				Name:         userInfo.Name,
				Email:        userInfo.Email,
				PasswordHash: "",
				IsVerified:   true,
			}
			err = s.userRepository.CreateUser(ctx, user)
			if err != nil {
				return nil, err
			}
		}

		err = s.userRepository.SaveOAuthConnection(ctx, &OAuthConnection{
			UserID:         user.ID,
			Provider:       provider,
			ProviderUserID: userInfo.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	accessToken, err := GenerateAccessToken(user, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshTokenHash := HashToken(refreshToken)

	err = s.userRepository.SaveRefreshToken(
		ctx,
		&RefreshToken{
			UserID:    user.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
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
