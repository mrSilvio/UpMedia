package domain

import "time"

type Recommendation struct {
	ID        ID
	ChannelID ID
	EventID   ID
	Score     float64
	Reasons   []string
	CreatedAt time.Time
}

type FeedbackValue string

const (
	FeedbackUseful        FeedbackValue = "useful"
	FeedbackNotInterested FeedbackValue = "not_interested"
)

type RecommendationFeedback struct {
	ID               ID
	RecommendationID ID
	EditorID         ID
	Value            FeedbackValue
	Comment          string
	CreatedAt        time.Time
}
