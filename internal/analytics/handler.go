package analytics

import (
	"net/http"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetAnalytics godoc
// @Summary      Get candidate telemetry and analytics
// @Description  Retrieves score trajectory trends, domain benchmarks, 90-day activity heatmaps, and pitfall insights over a given range.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        range query string false "Time range (7d, 30d, 90d, all, default: 30d)"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.AnalyticsResponse}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/analytics [get]
func (h *Handler) GetAnalytics(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	timeRange := c.DefaultQuery("range", "30d")

	data, err := h.service.GetAnalytics(c.Request.Context(), userID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
