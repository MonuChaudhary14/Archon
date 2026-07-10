package auth

import (
	"net/http"
	"strings"

	"github.com/MonuChaudhary14/Archon/pkg/response"
	"github.com/gin-gonic/gin"
)

// Register godoc
// @Summary Register a new user
// @Description Register a new user with name, email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration Info"
// @Success 201 {object} response.APIResponse{data=UserResponse}
// @Failure 400 {object} response.APIResponse
// @Router /register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	user, err := h.authService.Register(c.Request.Context(), req)

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "user registered successfully", user)
}

// VerifyEmail godoc
// @Summary Verify email with OTP
// @Description Verify email using the OTP sent to the user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "Verify Email Info"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /verify-email [post]
func (h *Handler) VerifyEmail(
	c *gin.Context,
) {

	var req VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.OTP = strings.TrimSpace(req.OTP)

	err := h.authService.VerifyEmail(
		c.Request.Context(),
		req,
	)

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "email verified successfully", nil)
}

// ResendOTP godoc
// @Summary Resend OTP
// @Description Resend OTP for email verification or password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResendOTPRequest true "Resend OTP Info"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /resend-otp [post]
func (h *Handler) ResendOTP(c *gin.Context) {
	var req ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Intent = strings.TrimSpace(req.Intent)

	if err := h.authService.ResendOTP(c.Request.Context(), req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "if the account is eligible, a new OTP has been sent", nil)
}
