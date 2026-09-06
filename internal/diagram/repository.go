package diagram

import (
	"context"
)

type Repository interface {
	SaveNode(ctx context.Context, n Node) error
	DeleteNode(ctx context.Context, interviewID string, id string) error
	SaveEdge(ctx context.Context, e Edge) error
	DeleteEdge(ctx context.Context, interviewID string, id string) error
	GetDiagram(ctx context.Context, interviewID string) ([]Node, []Edge, error)
}
