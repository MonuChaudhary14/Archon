package reports

import (
	"net/http"
	"strconv"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// ListReports godoc
// @Summary      List past interview reports
// @Description  Retrieves paginated and filtered interview reports for the authenticated candidate.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Search by title or summary"
// @Param        difficulty query string false "Filter by difficulty (beginner, intermediate, senior, staff)"
// @Param        page query int false "Page number (default: 1)"
// @Param        limit query int false "Items per page (default: 10)"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.ReportsListResponse}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/reports [get]
func (h *Handler) ListReports(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	search := c.Query("search")
	difficulty := c.Query("difficulty")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.ListReports(c.Request.Context(), userID, search, difficulty, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}

// GetReportDetail godoc
// @Summary      Get detailed evaluation report
// @Description  Retrieves multi-axis rubrics, strengths, weaknesses, and AI remarks for a session ID. Returns 202 Accepted if evaluation is in progress.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Interview Session ID"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.DetailedReportResponse}"
// @Success      202 {object} map[string]string "{"status": "evaluating"}"
// @Failure      400 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      404 {object} map[string]interface{} "{"success": false, "error": "report not found"}"
// @Router       /api/v1/reports/{id} [get]
func (h *Handler) GetReportDetail(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		sessionID = c.Param("session_id")
	}

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "session_id is required"})
		return
	}

	report, isEvaluating, err := h.service.GetReportDetail(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "report not found"})
		return
	}

	if isEvaluating {
		c.JSON(http.StatusAccepted, gin.H{
			"status": "evaluating",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}
