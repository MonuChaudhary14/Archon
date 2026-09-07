package diagram

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/interviews/:id/diagram", h.GetDiagram)
}
