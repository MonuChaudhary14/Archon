package diagram

import (
	"context"
	"net/http"

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
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: user ID not found in context"})
		return
	}

	userIDUint, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error: invalid user ID format"})
		return
	}
	userID := int(userIDUint)

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
