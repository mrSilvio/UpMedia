package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("event cluster not found")
	ErrAlreadyPublished = errors.New("event cluster is already published")
)

// VerificationStatus — 3 статуса проверки фактов по ТЗ
type VerificationStatus string

const (
	StatusConfirmed      VerificationStatus = "confirmed"         // «подтверждено»
	StatusContradictions VerificationStatus = "contradictions"    // «есть противоречия»
	StatusInsufficient   VerificationStatus = "insufficient_data" // «недостаточно данных»
)

// PostFormat — 3 целевых формата публикации ИИ-редактора
type PostFormat string

const (
	FormatUrgent   PostFormat = "urgent"   // Короткое срочное сообщение
	FormatDetailed PostFormat = "detailed" // Развернутый пост для Telegram
	FormatWeb      PostFormat = "web"      // Статья для сайта
)

// PublicationState — жизненный цикл карточки внутри редакции
type PublicationState string

const (
	StateDraft     PublicationState = "draft"
	StatePublished PublicationState = "published"
	StateRejected  PublicationState = "rejected"
)

// FeedbackType — оценка материала редактором для улучшения рекомендаций
type FeedbackType string

const (
	FeedbackUseful        FeedbackType = "useful"        // Полезный материал
	FeedbackUninteresting FeedbackType = "uninteresting" // Неинтересный материал
)

// Claim — конкретное утверждение, выделенное из новости
type Claim struct {
	Text        string
	IsVerified  bool
	EvidenceURL string
}

// EventCluster — единая карточка события (объединяет публикации об одном инфоповоде)
type EventCluster struct {
	ID                      string
	RawInput                string
	Summary                 string
	TopicTags               []string
	Entities                []string // Имена, гео-названия, ключевые слова (NER)
	PrimarySourceName       string   // Выявленный первоисточник
	PrimarySourceRating     float64  // Динамический рейтинг надежности источника (0..100)
	TotalPublications       int      // Общее число сообщений по теме
	ReprintsCount           int      // Из них перепечаток одного первоисточника
	IndependentConfirmCount int      // Независимых подтверждений
	Verification            VerificationStatus
	Claims                  []Claim
	Headlines               []string              // Варианты заголовков от ИИ
	SelectedHeadlineIdx     int                   // Выбранный заголовок
	Drafts                  map[PostFormat]string // Черновики под разные площадки
	ActiveFormat            PostFormat
	RelevanceScore          float64 // Персональный рейтинг для канала
	IsFromSuggestion        bool    // Изоляция пользовательской «предложки»
	Feedback                FeedbackType
	State                   PublicationState
	CreatedAt               time.Time
}

func (c *EventCluster) ActiveDraft() string {
	if text, ok := c.Drafts[c.ActiveFormat]; ok && text != "" {
		return text
	}
	return c.Summary
}

func (c *EventCluster) ActiveHeadline() string {
	if len(c.Headlines) == 0 {
		return "Без заголовка"
	}
	if c.SelectedHeadlineIdx >= 0 && c.SelectedHeadlineIdx < len(c.Headlines) {
		return c.Headlines[c.SelectedHeadlineIdx]
	}
	return c.Headlines[0]
}

// Контракты (Ports)

type ClusterRepository interface {
	Save(ctx context.Context, cluster *EventCluster) error
	GetByID(ctx context.Context, id string) (*EventCluster, error)
}

type AnalyzerLLM interface {
	AnalyzeEvent(ctx context.Context, rawText string, isSuggestion bool) (*EventCluster, error)
	VerifyDeeper(ctx context.Context, cluster *EventCluster) (*EventCluster, error)
}

type ChannelPublisher interface {
	PublishToChannel(ctx context.Context, formattedHTML string) error
}