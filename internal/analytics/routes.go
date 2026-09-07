package analytics

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/analytics", h.GetAnalytics)
}
