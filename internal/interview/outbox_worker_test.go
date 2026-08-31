package interview

import "testing"

func TestGetTopicForEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{"INTERVIEW_STARTED", "ai.requests"},
		{"INTERVIEW_SUBMITTED", "ai.requests"},
		{"UNKNOWN_EVENT", "ai.requests"},
	}

	for _, tt := range tests {
		actual := getTopicForEvent(tt.eventType)
		if actual != tt.expected {
			t.Errorf("getTopicForEvent(%q) = %q; expected %q", tt.eventType, actual, tt.expected)
		}
	}
}
