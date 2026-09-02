package dashboard

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

// GetOverview godoc
// @Summary      Get candidate dashboard overview
// @Description  Aggregates metrics, recommended mock scenario, competency radar, recent interviews, and syllabus roadmap.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.DashboardOverviewResponse}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/dashboard/overview [get]
func (h *Handler) GetOverview(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	data, err := h.service.GetOverview(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
