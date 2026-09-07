package settings

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/settings")
	{
		group.GET("", h.GetSettings)
		group.PUT("", h.UpdateSettings)
	}
}
