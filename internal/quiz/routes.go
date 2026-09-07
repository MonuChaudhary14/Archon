package quiz

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/quizzes")
	{
		group.GET("/daily-challenge", h.GetDailyChallenge)
		group.POST("/daily-challenge/verify", h.VerifyDailyChallenge)
		group.GET("/decks", h.ListDecks)
		group.GET("/decks/:id", h.GetDeckQuestions)
		group.POST("/decks/:id/submit", h.SubmitDeckQuiz)
	}
}
