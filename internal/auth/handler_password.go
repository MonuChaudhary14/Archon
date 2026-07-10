package auth

import (
	"net/http"
	"strings"

	"github.com/MonuChaudhary14/Archon/pkg/response"
	"github.com/gin-gonic/gin"
)

// ForgotPassword godoc
// @Summary Forgot password
// @Description Request a password reset email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Forgot Password Info"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	_ = h.authService.ForgotPassword(c.Request.Context(), req)

	response.Success(c, http.StatusOK, "if an account with that email exists, a password reset email has been sent", nil)
}

// VerifyResetOTP godoc
// @Summary Verify reset password OTP
// @Description Verify the OTP sent for password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VerifyResetOTPRequest true "Verify Reset OTP Info"
// @Success 200 {object} response.APIResponse{data=VerifyResetOTPResponse}
// @Failure 400 {object} response.APIResponse
// @Router /verify-reset-otp [post]
func (h *Handler) VerifyResetOTP(c *gin.Context) {
	var req VerifyResetOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.OTP = strings.TrimSpace(req.OTP)

	resp, err := h.authService.VerifyResetOTP(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "otp verified", resp)
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset Password Info"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.ResetToken = strings.TrimSpace(req.ResetToken)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if err := h.authService.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "password reset successfully", nil)
}
