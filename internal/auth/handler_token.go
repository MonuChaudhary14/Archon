package auth

import (
	"net/http"

	"github.com/MonuChaudhary14/Archon/pkg/response"
	"github.com/gin-gonic/gin"
)

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
