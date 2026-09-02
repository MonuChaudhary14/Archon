package models

type QuizOption struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	IsCorrect   bool   `json:"is_correct,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

type QuizQuestion struct {
	ID       string       `json:"id"`
	Question string       `json:"question"`
	Scenario string       `json:"scenario,omitempty"`
	TopicTag string       `json:"topic_tag"`
	Options  []QuizOption `json:"options"`
}

type QuizDeckItem struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Difficulty       string `json:"difficulty"`
	QuestionCount    int    `json:"question_count"`
	EstMinutes       int    `json:"est_minutes"`
	IconName         string `json:"icon_name"`
	Category         string `json:"category"`
	CompletedPercent int    `json:"completed_percent"`
}

type VerifyDailyChallengeRequest struct {
	QuestionID       string `json:"question_id" binding:"required"`
	SelectedOptionID string `json:"selected_option_id" binding:"required"`
}

type VerifyDailyChallengeResponse struct {
	IsCorrect       bool   `json:"is_correct"`
	CorrectOptionID string `json:"correct_option_id"`
	Explanation     string `json:"explanation"`
}

type SubmitDeckQuizRequest struct {
	TimeSpentSeconds int               `json:"time_spent_seconds"`
	Answers          map[string]string `json:"answers" binding:"required"`
}

type QuestionReviewItem struct {
	QuestionID      string `json:"question_id"`
	UserOptionID    string `json:"user_option_id"`
	CorrectOptionID string `json:"correct_option_id"`
	IsCorrect       bool   `json:"is_correct"`
	Explanation     string `json:"explanation"`
}

type SubmitDeckQuizResponse struct {
	DeckID         string               `json:"deck_id"`
	TotalQuestions int                  `json:"total_questions"`
	CorrectCount   int                  `json:"correct_count"`
	ScorePercent   int                  `json:"score_percent"`
	Review         []QuestionReviewItem `json:"review"`
}
