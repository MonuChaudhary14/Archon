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

// GetDailyChallenge godoc
// @Summary      Get today's daily architecture challenge
// @Description  Retrieves the active daily architecture challenge question without revealing correct answer.
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.QuizQuestion}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/quizzes/daily-challenge [get]
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

// VerifyDailyChallenge godoc
// @Summary      Verify daily challenge answer
// @Description  Verifies the candidate's selected option and returns the explanation and correctness.
// @Tags         quizzes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.VerifyDailyChallengeRequest true "Challenge verification payload"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.VerifyDailyChallengeResponse}"
// @Failure      400 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/quizzes/daily-challenge/verify [post]
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

// ListDecks godoc
// @Summary      List system design quiz decks
// @Description  Retrieves available practice topic decks with candidate mastery completion percentages.
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "{"success": true, "data": []models.QuizDeckItem}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/quizzes/decks [get]
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

// GetDeckQuestions godoc
// @Summary      Get questions for a quiz deck
// @Description  Retrieves all questions for a selected deck without revealing correct answers.
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Quiz Deck ID"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": []models.QuizQuestion}"
// @Failure      400 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/quizzes/decks/{id} [get]
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

// SubmitDeckQuiz godoc
// @Summary      Submit deck quiz answers
// @Description  Evaluates candidate deck answers, updates database mastery, and returns a detailed question review.
// @Tags         quizzes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Quiz Deck ID"
// @Param        request body models.SubmitDeckQuizRequest true "Submission payload"
// @Success      200 {object} map[string]interface{} "{"success": true, "data": models.SubmitDeckQuizResponse}"
// @Failure      400 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Failure      401 {object} map[string]interface{} "{"success": false, "error": "unauthorized"}"
// @Failure      500 {object} map[string]interface{} "{"success": false, "error": "..."}"
// @Router       /api/v1/quizzes/decks/{id}/submit [post]
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
