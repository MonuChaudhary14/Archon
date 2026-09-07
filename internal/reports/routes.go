package reports

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/reports")
	{
		group.GET("", h.ListReports)
		group.GET("/:id", h.GetReportDetail)
	}
}
