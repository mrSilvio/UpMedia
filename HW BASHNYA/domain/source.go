package domain

import "time"

type SourceType string

const (
	SourceTelegram SourceType = "telegram"
	SourceRSS      SourceType = "rss"
	SourceWebsite  SourceType = "website"
)

type Source struct {
	ID        ID
	Name      string
	URL       string
	Type      SourceType
	Enabled   bool
	Trust     TrustRating
	CreatedAt time.Time
}

type TrustRating struct {
	Evaluated      bool
	Score          float64
	ConfirmedCount int
	RefutedCount   int
	Explanation    string
	UpdatedAt      time.Time
}
