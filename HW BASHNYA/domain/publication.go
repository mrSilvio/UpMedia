package domain

import "time"

type PublicationFormat string

const (
	FormatShortPost PublicationFormat = "short_post"
	FormatLongPost  PublicationFormat = "long_post"
	FormatArticle   PublicationFormat = "article"
)

type PublicationPlatform string

const (
	PlatformTelegram PublicationPlatform = "telegram"
	PlatformWebsite  PublicationPlatform = "website"
)

type DraftStatus string

const (
	DraftEditing   DraftStatus = "editing"
	DraftApproved  DraftStatus = "approved"
	DraftPublished DraftStatus = "published"
	DraftRejected  DraftStatus = "rejected"
)

type PublicationDraft struct {
	ID                  ID
	EventID             ID
	ChannelID           ID
	AuthorID            ID
	Format              PublicationFormat
	Platform            PublicationPlatform
	Tone                string
	HeadlineVariants    []string
	SelectedHeadline    string
	Text                string
	SourceURLs          []string
	UnconfirmedClaimIDs []ID
	Status              DraftStatus
	ApprovedBy          ID
	ApprovedAt          *time.Time
	PublishedAt         *time.Time
	PublishedURL        string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
