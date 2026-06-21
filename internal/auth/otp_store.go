package auth

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type OTPStore struct {
	client *redis.Client
}

func NewOTPStore(client *redis.Client) *OTPStore {

	return &OTPStore{
		client: client,
	}
}

func (s *OTPStore) SaveVerificationOTP(ctx context.Context, email string, otpHash string) error {

	key := "verify:" + email

	return s.client.Set(
		ctx,
		key,
		otpHash,
		10*time.Minute,
	).Err()
}

func (s *OTPStore) GetVerificationOTP(ctx context.Context, email string) (string, error) {

	key := "verify:" + email

	return s.client.Get(
		ctx,
		key,
	).Result()
}

func (s *OTPStore) DeleteVerificationOTP(ctx context.Context, email string) error {

	key := "verify:" + email

	return s.client.Del(
		ctx,
		key,
	).Err()
}

func (s *OTPStore) SavePasswordResetOTP(ctx context.Context, email string, otpHash string) error {
	key := "reset:" + email
	return s.client.Set(ctx, key, otpHash, 15*time.Minute).Err()
}

func (s *OTPStore) GetPasswordResetOTP(ctx context.Context, email string) (string, error) {
	key := "reset:" + email
	return s.client.Get(ctx, key).Result()
}

func (s *OTPStore) DeletePasswordResetOTP(ctx context.Context, email string) error {
	key := "reset:" + email
	return s.client.Del(ctx, key).Err()
}
