package auth

type EmailJob struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}
