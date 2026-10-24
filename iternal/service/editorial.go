package service

import (
	"context"
	"fmt"
	"sync"

	"upmedia/internal/domain"
)

type EditorialService struct {
	repo      domain.ClusterRepository
	llm       domain.AnalyzerLLM
	publisher domain.ChannelPublisher
	mu        sync.Mutex
}

func NewEditorialService(
	repo domain.ClusterRepository,
	llm domain.AnalyzerLLM,
	pub domain.ChannelPublisher,
) *EditorialService {
	return &EditorialService{
		repo:      repo,
		llm:       llm,
		publisher: pub,
	}
}

// ProcessIncoming обрабатывает новый инфоповод (из парсера или из «предложки»)
func (s *EditorialService) ProcessIncoming(ctx context.Context, rawText string, isSuggestion bool) (*domain.EventCluster, error) {
	cluster, err := s.llm.AnalyzeEvent(ctx, rawText, isSuggestion)
	if err != nil {
		return nil, fmt.Errorf("analyze event: %w", err)
	}

	// По ТЗ: непроверенные сообщения из предложки не получают статус достоверных автоматически
	if isSuggestion && cluster.IndependentConfirmCount == 0 {
		cluster.Verification = domain.StatusInsufficient
	}

	cluster.State = domain.StateDraft
	cluster.ActiveFormat = domain.FormatUrgent

	if err := s.repo.Save(ctx, cluster); err != nil {
		return nil, fmt.Errorf("save cluster: %w", err)
	}
	return cluster, nil
}

func (s *EditorialService) GetCluster(ctx context.Context, id string) (*domain.EventCluster, error) {
	return s.repo.GetByID(ctx, id)
}

// SwitchFormat переключает формат: срочное сообщение / развернутый пост / статья для сайта
func (s *EditorialService) SwitchFormat(ctx context.Context, id string, format domain.PostFormat) (*domain.EventCluster, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cluster.ActiveFormat = format
	if err := s.repo.Save(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// CycleHeadline переключает предложенные ИИ варианты заголовков
func (s *EditorialService) CycleHeadline(ctx context.Context, id string) (*domain.EventCluster, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(cluster.Headlines) > 0 {
		cluster.SelectedHeadlineIdx = (cluster.SelectedHeadlineIdx + 1) % len(cluster.Headlines)
	}
	if err := s.repo.Save(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// RequestAdditionalCheck запускает углубленную перепроверку фактов и первоисточников
func (s *EditorialService) RequestAdditionalCheck(ctx context.Context, id string) (*domain.EventCluster, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.llm.VerifyDeeper(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("deeper verification: %w", err)
	}

	if err := s.repo.Save(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// RecordFeedback сохраняет оценку редактора (полезно / неинтересно) для обучения рекомендаций
func (s *EditorialService) RecordFeedback(ctx context.Context, id string, fb domain.FeedbackType) (*domain.EventCluster, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cluster.Feedback = fb
	if fb == domain.FeedbackUninteresting {
		cluster.State = domain.StateRejected
	}
	if err := s.repo.Save(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Publish публикует выбранный вариант в целевой Telegram-канал
func (s *EditorialService) Publish(ctx context.Context, id string, formattedHTML string) (*domain.EventCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cluster.State == domain.StatePublished {
		return nil, domain.ErrAlreadyPublished
	}

	if err := s.publisher.PublishToChannel(ctx, formattedHTML); err != nil {
		return nil, fmt.Errorf("publish to channel: %w", err)
	}

	cluster.State = domain.StatePublished
	if err := s.repo.Save(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}