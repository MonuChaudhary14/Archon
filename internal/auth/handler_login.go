package auth

import (
	"net/http"
	"strings"

	"github.com/MonuChaudhary14/Archon/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

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

	frontendURLs := strings.Split(h.frontendURL, ",")
	redirectURL := frontendURLs[0]
	if redirectURL == "" {
		redirectURL = "http://localhost:3000"
	}

	c.Redirect(http.StatusFound, redirectURL+"/dashboard")
}
