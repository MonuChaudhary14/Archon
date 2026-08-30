package diagram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockRepository struct {
	nodes []Node
	edges []Edge
	err   error
}

func (m *mockRepository) SaveNode(ctx context.Context, n Node) error {
	return nil
}
func (m *mockRepository) DeleteNode(ctx context.Context, interviewID string, id string) error {
	return nil
}
func (m *mockRepository) SaveEdge(ctx context.Context, e Edge) error {
	return nil
}
func (m *mockRepository) DeleteEdge(ctx context.Context, interviewID string, id string) error {
	return nil
}
func (m *mockRepository) GetDiagram(ctx context.Context, interviewID string) ([]Node, []Edge, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.nodes, m.edges, nil
}

func TestGetDiagram(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         interface{}
		paramID        string
		verifyOwnerRes bool
		verifyOwnerErr error
		repoNodes      []Node
		repoEdges      []Edge
		repoErr        error
		expectedStatus int
		verifyJSON     func(t *testing.T, body string)
	}{
		{
			name:           "Success Retrieval",
			userID:         uint(42),
			paramID:        "interview-123",
			verifyOwnerRes: true,
			repoNodes:      []Node{{ID: "node1", Label: "Cache", Type: "cache"}},
			repoEdges:      []Edge{{ID: "edge1", Source: "user", Target: "node1"}},
			expectedStatus: http.StatusOK,
			verifyJSON: func(t *testing.T, body string) {
				var resp DiagramResponse
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Nodes) != 1 || resp.Nodes[0].ID != "node1" {
					t.Errorf("expected 1 node with ID 'node1', got: %+v", resp.Nodes)
				}
				if len(resp.Edges) != 1 || resp.Edges[0].ID != "edge1" {
					t.Errorf("expected 1 edge with ID 'edge1', got: %+v", resp.Edges)
				}
			},
		},
		{
			name:           "Unauthorized - No UserID in Context",
			userID:         nil,
			paramID:        "interview-123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden - Not Owner",
			userID:         uint(42),
			paramID:        "interview-123",
			verifyOwnerRes: false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Internal Error - Repository Failure",
			userID:         uint(42),
			paramID:        "interview-123",
			verifyOwnerRes: true,
			repoErr:        errors.New("db disconnect"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				nodes: tt.repoNodes,
				edges: tt.repoEdges,
				err:   tt.repoErr,
			}

			verifyOwner := func(ctx context.Context, userID int, interviewID string) (bool, error) {
				return tt.verifyOwnerRes, tt.verifyOwnerErr
			}

			handler := NewHandler(repo, verifyOwner)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/api/v1/interviews/:id/diagram", func(ctx *gin.Context) {
				if tt.userID != nil {
					ctx.Set("userID", tt.userID)
				}
				handler.GetDiagram(ctx)
			})

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+tt.paramID+"/diagram", nil)
			c.Request = req

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.verifyJSON != nil {
				tt.verifyJSON(t, w.Body.String())
			}
		})
	}
}
