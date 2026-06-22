package auth

import (
	"net/http"

	"github.com/MonuChaudhary14/sys/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService AuthService
}

func NewHandler(
	authService AuthService,
) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req)

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "user registered successfully", user)
}

func (h *Handler) VerifyEmail(
	c *gin.Context,
) {

	var req VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

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

func (h *Handler) Login(
	c *gin.Context,
) {

	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authService.Login(
		c.Request.Context(),
		req,
	)

	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("access_token", resp.AccessToken, 15*60, "/", "", true, true)
	c.SetCookie("refresh_token", resp.RefreshToken, 7*24*60*60, "/", "", true, true)

	response.Success(c, http.StatusOK, "logged in successfully", resp)
}

func (h *Handler) Refresh(
	c *gin.Context,
) {

	var req RefreshRequest

	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		req.RefreshToken = refreshToken
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "refresh token not found in cookie or request body")
			return
		}
	}

	resp, err := h.authService.Refresh(
		c.Request.Context(),
		req,
	)

	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("access_token", resp.AccessToken, 15*60, "/", "", true, true)
	c.SetCookie("refresh_token", resp.RefreshToken, 7*24*60*60, "/", "", true, true)

	response.Success(c, http.StatusOK, "token refreshed successfully", resp)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest

	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		req.RefreshToken = refreshToken
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "refresh token not found in cookie or request body")
			return
		}
	}

	if err := h.authService.Logout(c.Request.Context(), req); err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("access_token", "", -1, "/", "", true, true)
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	response.Success(c, http.StatusOK, "logged out successfully", nil)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = h.authService.ForgotPassword(c.Request.Context(), req)

	response.Success(c, http.StatusOK, "if an account with that email exists, a password reset email has been sent", nil)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "password reset successfully", nil)
}

