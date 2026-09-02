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
