package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"upmedia/internal/adapter/telegram"
	"upmedia/internal/domain"
	"upmedia/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	channel := os.Getenv("TELEGRAM_CHANNEL_ID")
	if token == "" || channel == "" {
		logger.Error("TELEGRAM_BOT_TOKEN and TELEGRAM_CHANNEL_ID are required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := NewMemoryClusterRepo()
	llmClient := NewMockAnalyzer()
	presenter := telegram.NewPresenter()

	botAdapter, err := telegram.NewBotAdapter(ctx, telegram.Config{
		Token:          token,
		TargetChannel:  channel,
		HandlerTimeout: 30 * time.Second,
	}, presenter, logger)
	if err != nil {
		logger.Error("init bot adapter failed", slog.Any("err", err))
		os.Exit(1)
	}

	editorialSvc := service.NewEditorialService(repo, llmClient, botAdapter)
	botAdapter.RegisterHandlers(editorialSvc)

	go func() {
		logger.Info("UpMedia bot started", slog.String("channel", channel))
		botAdapter.Start()
	}()

	<-ctx.Done()
	logger.Info("stopping bot gracefully...")
	botAdapter.Stop()
}

// --- In-Memory Репозиторий ---

type MemoryClusterRepo struct {
	mu    sync.RWMutex
	store map[string]*domain.EventCluster
}

func NewMemoryClusterRepo() *MemoryClusterRepo {
	return &MemoryClusterRepo{store: make(map[string]*domain.EventCluster)}
}

func (r *MemoryClusterRepo) Save(_ context.Context, c *domain.EventCluster) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := *c
	r.store[c.ID] = &clone
	return nil
}

func (r *MemoryClusterRepo) GetByID(_ context.Context, id string) (*domain.EventCluster, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.store[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *c
	return &clone, nil
}

// --- Mock-реализация анализатора (демонстрирует все сценарии из UpMedia_Plan.pdf) ---

type MockAnalyzer struct{}

func NewMockAnalyzer() *MockAnalyzer { return &MockAnalyzer{} }

func (m *MockAnalyzer) AnalyzeEvent(_ context.Context, rawText string, isSuggestion bool) (*domain.EventCluster, error) {
	id := fmt.Sprintf("evt_%d", time.Now().UnixNano())

	return &domain.EventCluster{
		ID:                      id,
		RawInput:                rawText,
		Summary:                 rawText,
		TopicTags:               []string{"#медиа", "#технологии", "#инфоповод"},
		Entities:                []string{"Россия", "ИИ-редакция", "Telegram"},
		PrimarySourceName:       "ТАСС / Официальный релиз",
		PrimarySourceRating:     92.5,
		TotalPublications:       7,
		ReprintsCount:           5, // 5 каналов просто перепечатали 1 источник
		IndependentConfirmCount: 2, // 2 независимых подтверждения
		Verification:            domain.StatusContradictions,
		Claims: []domain.Claim{
			{
				Text:        "Основной факт события зафиксирован в первоисточнике",
				IsVerified:  true,
				EvidenceURL: "https://tass.ru",
			},
			{
				Text:        "Данные о масштабах и точных сроках различаются в перепечатках",
				IsVerified:  false,
				EvidenceURL: "",
			},
		},
		Headlines: []string{
			"Молния: ключевые подробности нового события",
			"Что известно к этому часу: разбор ситуации и фактов",
			"Фактчекинг события: какие детали подтвердились официально",
		},
		Drafts: map[domain.PostFormat]string{
			domain.FormatUrgent: "⚡ СРОЧНО: " + rawText + "\n\n⚠️ Примечание: часть второстепенных деталей пока не имеет независимых подтверждений.",
			domain.FormatDetailed: "📊 Подробный разбор инфоповода:\n\n" + rawText +
				"\n\nСистема проанализировала 7 публикаций (из них 5 являются перепечатками). Основной тезис подтвержден, однако точные цифры еще уточняются.",
			domain.FormatWeb: "Статья для сайта:\n\nВведение и контекст:\n" + rawText +
				"\n\nАнализ источников:\nПо данным верифицированных агентств, информация частично подтверждена. Неподтвержденные утверждения очевидцев выделены в отдельный блок.",
		},
		IsFromSuggestion: isSuggestion,
		CreatedAt:        time.Now(),
	}, nil
}

func (m *MockAnalyzer) VerifyDeeper(_ context.Context, c *domain.EventCluster) (*domain.EventCluster, error) {
	c.IndependentConfirmCount++
	c.Verification = domain.StatusConfirmed
	for i := range c.Claims {
		c.Claims[i].IsVerified = true
		if c.Claims[i].EvidenceURL == "" {
			c.Claims[i].EvidenceURL = "https://rbc.ru"
		}
	}
	return c, nil
}