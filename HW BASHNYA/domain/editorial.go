package domain

import "time"

type Editor struct {
	ID         ID
	TelegramID int64
	Name       string
	CreatedAt  time.Time
}

type Channel struct {
	ID         ID
	Title      string
	TelegramID int64
	EditorIDs  []ID
	Profile    ChannelProfile
	CreatedAt  time.Time
}

type ChannelProfile struct {
	Topics           []Tag
	Regions          []string
	PreferredTone    string
	PreferredFormats []PublicationFormat
}
