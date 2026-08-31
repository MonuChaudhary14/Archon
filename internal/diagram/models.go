package diagram

type Node struct {
	ID          string  `json:"id"`
	InterviewID string  `json:"interview_id"`
	Type        string  `json:"type"`
	Label       string  `json:"label"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

type Edge struct {
	ID          string `json:"id"`
	InterviewID string `json:"interview_id"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"`
}

type DiagramResponse struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
