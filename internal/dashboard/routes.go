package dashboard

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/dashboard/overview", h.GetOverview)
}
