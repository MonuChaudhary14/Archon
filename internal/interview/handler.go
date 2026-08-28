package interview

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// StartInterview godoc
// @Summary      Start a new system design interview
// @Description  Creates a new system design interview session and returns the initial question.
// @Tags         interviews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateInterviewRequest true "Interview Setup Details"
// @Success      200 {object} StartInterviewResponse
// @Failure      400 {object} map[string]string "{"error": "..."}"
// @Failure      401 {object} map[string]string "{"error": "..."}"
// @Failure      500 {object} map[string]string "{"error": "..."}"
// @Router       /api/v1/interviews/start [post]
func (h *Handler) StartInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: user ID not found in context"})
		return
	}

	userIDUint, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error: invalid user ID format"})
		return
	}
	userID := int(userIDUint)

	var req CreateInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request, 'difficulty' is required"})
		return
	}

	question, sessionID, err := h.service.StartInterview(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"question":   question,
	})
}

// GetInterviewReport godoc
// @Summary      Get the report of a completed interview
// @Description  Retrieves the evaluation report, score, and feedback of an interview session by ID.
// @Tags         interviews
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Interview Session ID"
// @Success      200 {object} Interview
// @Success      202 {object} map[string]string "{"status": "evaluation in progress"}"
// @Failure      400 {object} map[string]string "{"error": "..."}"
// @Failure      401 {object} map[string]string "{"error": "..."}"
// @Failure      404 {object} map[string]string "{"error": "..."}"
// @Router       /api/v1/interviews/{id}/report [get]
func (h *Handler) GetInterviewReport(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: user ID not found in context"})
		return
	}

	userIDUint, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error: invalid user ID format"})
		return
	}
	userID := int(userIDUint)

	interviewID := c.Param("id")
	if interviewID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interview ID is required"})
		return
	}

	interview, err := h.service.GetInterviewByID(c.Request.Context(), userID, interviewID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "interview not found or unauthorized"})
		return
	}

	if interview.Score == nil {
		c.JSON(http.StatusAccepted, gin.H{"status": "evaluation in progress"})
		return
	}

	c.JSON(http.StatusOK, interview)
}
