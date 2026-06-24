package auth

import (
	"net/http"
	"strings"

	"github.com/MonuChaudhary14/sys/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type Handler struct {
	authService    AuthService
	oauthProviders map[string]OAuthProvider
	frontendURL    string
}

func NewHandler(
	authService AuthService,
	oauthProviders map[string]OAuthProvider,
	frontendURL string,
) *Handler {
	return &Handler{
		authService:    authService,
		oauthProviders: oauthProviders,
		frontendURL:    frontendURL,
	}
}

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

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Info"
// @Success 200 {object} response.APIResponse{data=LoginResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Router /login [post]
func (h *Handler) Login(
	c *gin.Context,
) {

	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

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

// OAuthLoginInitiate godoc
// @Summary Initiate OAuth Login
// @Description Redirects the user to the specified OAuth provider's login page
// @Tags auth
// @Param provider path string true "OAuth Provider (google or github)"
// @Success 307 "Temporary Redirect to OAuth Provider"
// @Failure 400 {object} response.APIResponse
// @Router /login/{provider} [get]
func (h *Handler) OAuthLoginInitiate(c *gin.Context) {
	providerName := c.Param("provider")
	provider, exists := h.oauthProviders[providerName]
	if !exists {
		response.Error(c, http.StatusBadRequest, "unsupported oauth provider")
		return
	}

	url := provider.GetConfig().AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuthCallback godoc
// @Summary OAuth Callback Handler
// @Description Handles the callback from the OAuth provider, exchanges the code for tokens, logs the user in, and redirects to the frontend dashboard.
// @Tags auth
// @Param provider path string true "OAuth Provider (google or github)"
// @Param code query string true "Authorization Code from provider"
// @Success 302 "Redirect to Frontend Dashboard"
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Router /callback/{provider} [get]
func (h *Handler) OAuthCallback(c *gin.Context) {
	providerName := c.Param("provider")
	provider, exists := h.oauthProviders[providerName]
	if !exists {
		response.Error(c, http.StatusBadRequest, "unsupported oauth provider")
		return
	}

	code := c.Query("code")
	if code == "" {
		response.Error(c, http.StatusBadRequest, "code not found")
		return
	}

	token, err := provider.GetConfig().Exchange(c.Request.Context(), code)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "failed to exchange token")
		return
	}

	userInfo, err := provider.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "failed to get user info")
		return
	}

	resp, err := h.authService.OAuthLogin(c.Request.Context(), providerName, userInfo)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("access_token", resp.AccessToken, 15*60, "/", "", true, true)
	c.SetCookie("refresh_token", resp.RefreshToken, 7*24*60*60, "/", "", true, true)

	c.Redirect(http.StatusFound, h.frontendURL+"/dashboard")
}

// Refresh godoc
// @Summary Refresh token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest false "Refresh Token Info"
// @Success 200 {object} response.APIResponse{data=LoginResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Router /refresh [post]
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

// Logout godoc
// @Summary Logout user
// @Description Logout user and invalidate refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest false "Logout Info"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /logout [post]
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
