package domain

import "time"

type NewsItem struct {
	ID          ID
	SourceID    ID
	ExternalID  string
	URL         string
	OriginalURL string
	Title       string
	Text        string
	Language    string
	Tags        []Tag
	Attachments []Attachment
	EventID     ID
	PublishedAt time.Time
	CollectedAt time.Time
}

type Event struct {
	ID            ID
	Title         string
	Summary       string
	Tags          []Tag
	NewsItemIDs   []ID
	SubmissionIDs []ID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
