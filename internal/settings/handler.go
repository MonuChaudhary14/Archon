package settings

import (
	"net/http"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/MonuChaudhary14/Archon/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetSettings godoc
// @Summary      Get candidate settings and preferences
// @Description  Retrieves workspace, role target, AI interviewer persona, and canvas grid configuration.
// @Tags         settings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.UserSettings}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	data, err := h.service.GetSettings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// UpdateSettings godoc
// @Summary      Update candidate settings and preferences
// @Description  Upserts workspace preferences, target companies, AI persona strictness, and canvas defaults.
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.UserSettings true "Settings update payload"
// @Success      200 {object} map[string]interface{} "{"success": true, "message": "Settings updated successfully"}"
// @Failure      400 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	var req models.UserSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid settings payload"})
		return
	}

	err := h.service.UpdateSettings(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Settings updated successfully",
	})
}
