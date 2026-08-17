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
