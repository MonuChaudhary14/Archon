package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateOTP() (string, error) {

	bytes := make([]byte, 3)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	number :=
		int(bytes[0])<<16 |
			int(bytes[1])<<8 |
			int(bytes[2])

	otp := fmt.Sprintf(
		"%06d",
		number%1000000,
	)

	return otp, nil
}

func HashOTP(otp string) string {

	hash := sha256.Sum256(
		[]byte(otp),
	)

	return hex.EncodeToString(
		hash[:],
	)
}
