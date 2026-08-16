package diagram

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SaveNode(ctx context.Context, n Node) error
	DeleteNode(ctx context.Context, interviewID string, id string) error
	SaveEdge(ctx context.Context, e Edge) error
	DeleteEdge(ctx context.Context, interviewID string, id string) error
	GetDiagram(ctx context.Context, interviewID string) ([]Node, []Edge, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) SaveNode(ctx context.Context, n Node) error {
	query := `
		INSERT INTO diagram_nodes (id, interview_id, type, label, x, y, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			label = EXCLUDED.label,
			x = EXCLUDED.x,
			y = EXCLUDED.y,
			updated_at = NOW();
	`
	_, err := r.db.Exec(ctx, query, n.ID, n.InterviewID, n.Type, n.Label, n.X, n.Y)
	if err != nil {
		return fmt.Errorf("failed to save diagram node: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteNode(ctx context.Context, interviewID string, id string) error {
	query := `
		DELETE FROM diagram_nodes 
		WHERE id = $1 AND interview_id = $2;
	`
	_, err := r.db.Exec(ctx, query, id, interviewID)
	if err != nil {
		return fmt.Errorf("failed to delete diagram node: %w", err)
	}
	return nil
}

func (r *postgresRepository) SaveEdge(ctx context.Context, e Edge) error {
	query := `
		INSERT INTO diagram_edges (id, interview_id, source, target, type, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (id) DO UPDATE SET
			source = EXCLUDED.source,
			target = EXCLUDED.target,
			type = EXCLUDED.type,
			updated_at = NOW();
	`
	_, err := r.db.Exec(ctx, query, e.ID, e.InterviewID, e.Source, e.Target, e.Type)
	if err != nil {
		return fmt.Errorf("failed to save diagram edge: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteEdge(ctx context.Context, interviewID string, id string) error {
	query := `
		DELETE FROM diagram_edges 
		WHERE id = $1 AND interview_id = $2;
	`
	_, err := r.db.Exec(ctx, query, id, interviewID)
	if err != nil {
		return fmt.Errorf("failed to delete diagram edge: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetDiagram(ctx context.Context, interviewID string) ([]Node, []Edge, error) {
	nodeQuery := `
		SELECT id, interview_id, type, label, x, y 
		FROM diagram_nodes 
		WHERE interview_id = $1;
	`
	nodes := []Node{}
	rows, err := r.db.Query(ctx, nodeQuery, interviewID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query diagram nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.InterviewID, &n.Type, &n.Label, &n.X, &n.Y); err != nil {
			return nil, nil, fmt.Errorf("failed to scan diagram node: %w", err)
		}
		nodes = append(nodes, n)
	}

	edgeQuery := `
		SELECT id, interview_id, source, target, type 
		FROM diagram_edges 
		WHERE interview_id = $1;
	`
	edges := []Edge{}
	rows2, err := r.db.Query(ctx, edgeQuery, interviewID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query diagram edges: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var e Edge
		if err := rows2.Scan(&e.ID, &e.InterviewID, &e.Source, &e.Target, &e.Type); err != nil {
			return nil, nil, fmt.Errorf("failed to scan diagram edge: %w", err)
		}
		edges = append(edges, e)
	}

	return nodes, edges, nil
}
