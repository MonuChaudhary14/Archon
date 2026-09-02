package quiz

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

func (h *Handler) GetDailyChallenge(c *gin.Context) {
	q, err := h.service.GetDailyChallenge(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    q,
	})
}

func (h *Handler) VerifyDailyChallenge(c *gin.Context) {
	var req models.VerifyDailyChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}

	res, err := h.service.VerifyDailyChallenge(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}

func (h *Handler) ListDecks(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	decks, err := h.service.ListDecks(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    decks,
	})
}

func (h *Handler) GetDeckQuestions(c *gin.Context) {
	deckID := c.Param("id")
	if deckID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "deck id is required"})
		return
	}

	questions, err := h.service.GetDeckQuestions(c.Request.Context(), deckID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    questions,
	})
}

func (h *Handler) SubmitDeckQuiz(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return
	}

	deckID := c.Param("id")
	if deckID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "deck id is required"})
		return
	}

	var req models.SubmitDeckQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid answers payload"})
		return
	}

	res, err := h.service.SubmitDeckQuiz(c.Request.Context(), userID, deckID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}
