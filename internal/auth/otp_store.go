package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
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

func (s *OTPStore) SaveUnverifiedUser(ctx context.Context, email string, user *UnverifiedUser) error {
	key := "unverified:" + email
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, 10*time.Minute).Err()
}

func (s *OTPStore) GetUnverifiedUser(ctx context.Context, email string) (*UnverifiedUser, error) {
	key := "unverified:" + email
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var user UnverifiedUser
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *OTPStore) DeleteUnverifiedUser(ctx context.Context, email string) error {
	key := "unverified:" + email
	return s.client.Del(ctx, key).Err()
}

func (s *OTPStore) SetOTPCooldown(ctx context.Context, email string) error {
	key := "otp_cooldown:" + email
	return s.client.Set(ctx, key, "1", 60*time.Second).Err()
}

func (s *OTPStore) CheckOTPCooldown(ctx context.Context, email string) bool {
	key := "otp_cooldown:" + email
	res, err := s.client.Exists(ctx, key).Result()
	return err == nil && res > 0
}

func (s *OTPStore) SaveResetToken(ctx context.Context, token, email string) error {
	key := "reset_token:" + token
	return s.client.Set(ctx, key, email, 10*time.Minute).Err()
}

func (s *OTPStore) GetEmailByResetToken(ctx context.Context, token string) (string, error) {
	key := "reset_token:" + token
	return s.client.Get(ctx, key).Result()
}

func (s *OTPStore) DeleteResetToken(ctx context.Context, token string) error {
	key := "reset_token:" + token
	return s.client.Del(ctx, key).Err()
}
