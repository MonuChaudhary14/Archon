package diagram

import (
	"context"
	"net/http"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo        Repository
	verifyOwner func(ctx context.Context, userID int, interviewID string) (bool, error)
}

func NewHandler(repo Repository, verifyOwner func(ctx context.Context, userID int, interviewID string) (bool, error)) *Handler {
	return &Handler{
		repo:        repo,
		verifyOwner: verifyOwner,
	}
}

// GetDiagram godoc
// @Summary      Get diagram state
// @Description  Retrieves the diagram nodes and edges for a specific interview session.
// @Tags         diagrams
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Interview Session ID"
// @Success      200 {object} DiagramResponse
// @Failure      400 {object} map[string]string "{"error": "..."}"
// @Failure      401 {object} map[string]string "{"error": "..."}"
// @Failure      403 {object} map[string]string "{"error": "..."}"
// @Failure      500 {object} map[string]string "{"error": "..."}"
// @Router       /api/v1/interviews/{id}/diagram [get]
func (h *Handler) GetDiagram(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return
	}

	interviewID := c.Param("id")
	if interviewID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interview ID is required"})
		return
	}

	isOwner, err := h.verifyOwner(c.Request.Context(), userID, interviewID)
	if err != nil || !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: you do not have access to this interview session"})
		return
	}

	nodes, edges, err := h.repo.GetDiagram(c.Request.Context(), interviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, DiagramResponse{
		Nodes: nodes,
		Edges: edges,
	})
}
