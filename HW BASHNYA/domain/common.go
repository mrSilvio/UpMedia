package domain

import "time"

type ID string

type Tag string

type Attachment struct {
	URL      string
	MIMEType string
	Caption  string
}

type Evidence struct {
	URL         string
	OriginalURL string
	SourceID    ID
	Quote       string
	FoundAt     time.Time
}
