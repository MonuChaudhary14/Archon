package interview

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/interviews")
	{
		group.GET("", h.ListInterviews)
		group.POST("/start", h.StartInterview)
		group.GET("/questions", h.ListQuestions)
		group.GET("/:id/report", h.GetInterviewReport)
		group.POST("/:id/submit", h.SubmitInterview)
	}
}
