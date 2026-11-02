package domain

import "time"

type SubmissionStatus string

const (
	SubmissionNew      SubmissionStatus = "new"
	SubmissionInReview SubmissionStatus = "in_review"
	SubmissionAccepted SubmissionStatus = "accepted"
	SubmissionRejected SubmissionStatus = "rejected"
)

type Submission struct {
	ID               ID
	ChannelID        ID
	SenderTelegramID int64
	Text             string
	Links            []string
	Attachments      []Attachment
	Status           SubmissionStatus
	ReviewerID       ID
	ReviewComment    string
	CreatedAt        time.Time
	ReviewedAt       *time.Time
}
