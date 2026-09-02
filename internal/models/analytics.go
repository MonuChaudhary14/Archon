package models

type TrendPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type DomainMastery struct {
	Domain         string `json:"domain"`
	Score          int    `json:"score"`
	Benchmark      int    `json:"benchmark"`
	QuestionsCount int    `json:"questions_count"`
}

type HeatmapDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type PitfallInsight struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Frequency   string `json:"frequency"`
}

type AnalyticsResponse struct {
	ReadinessScore    int              `json:"readiness_score"`
	Percentile        int              `json:"percentile"`
	TotalHoursTrained float64          `json:"total_hours_trained"`
	AvgScore          int              `json:"avg_score"`
	TargetLevel       string           `json:"target_level"`
	Trend             []TrendPoint     `json:"trend"`
	DomainMastery     []DomainMastery  `json:"domain_mastery"`
	HeatmapDays       []HeatmapDay     `json:"heatmap_days"`
	Pitfalls          []PitfallInsight `json:"pitfalls"`
}
