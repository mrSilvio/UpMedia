package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
	"upmedia/internal/domain"
	"upmedia/internal/service"
)

type BotAdapter struct {
	baseCtx        context.Context
	bot            *tele.Bot
	service        *service.EditorialService
	presenter      *Presenter
	logger         *slog.Logger
	targetChannel  string
	handlerTimeout time.Duration
}

type Config struct {
	Token          string
	TargetChannel  string
	AllowedEditors map[int64]struct{}
	HandlerTimeout time.Duration
}

func NewBotAdapter(
	baseCtx context.Context,
	cfg Config,
	presenter *Presenter,
	logger *slog.Logger,
) (*BotAdapter, error) {
	pref := tele.Settings{
		Token:     cfg.Token,
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeHTML,
		OnError: func(err error, c tele.Context) {
			logger.Error("telebot error", slog.Any("err", err))
		},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("init telebot: %w", err)
	}

	b.Use(RecoverMiddleware(logger))
	b.Use(SlogMiddleware(logger))
	b.Use(EditorAuthMiddleware(cfg.AllowedEditors))

	return &BotAdapter{
		baseCtx:        baseCtx,
		bot:            b,
		presenter:      presenter,
		logger:         logger,
		targetChannel:  cfg.TargetChannel,
		handlerTimeout: cfg.HandlerTimeout,
	}, nil
}

func (a *BotAdapter) RegisterHandlers(svc *service.EditorialService) {
	a.service = svc

	a.bot.Handle("/start", a.handleStart)
	a.bot.Handle("/suggest", a.handleSuggestionCommand)
	a.bot.Handle(tele.OnText, a.handleIncomingText)
	a.bot.Handle(tele.OnPhoto, a.handleIncomingPhoto) // Поддержка фото в предложке

	// Инлайн-кнопки карточки редактора
	a.bot.Handle(&BtnFormatUrgent, a.handleSwitchFormat(domain.FormatUrgent))
	a.bot.Handle(&BtnFormatDetailed, a.handleSwitchFormat(domain.FormatDetailed))
	a.bot.Handle(&BtnFormatWeb, a.handleSwitchFormat(domain.FormatWeb))
	a.bot.Handle(&BtnCycleHeadline, a.handleCycleHeadline)
	a.bot.Handle(&BtnRecheck, a.handleRecheck)
	a.bot.Handle(&BtnFeedbackLike, a.handleFeedback(domain.FeedbackUseful))
	a.bot.Handle(&BtnFeedbackSkip, a.handleFeedback(domain.FeedbackUninteresting))
	a.bot.Handle(&BtnPublish, a.handlePublish)
}

func (a *BotAdapter) PublishToChannel(_ context.Context, formattedHTML string) error {
	recipient := tele.ChatUsername(a.targetChannel)
	_, err := a.bot.Send(recipient, formattedHTML, tele.ModeHTML)
	return err
}

func (a *BotAdapter) Start() { a.bot.Start() }
func (a *BotAdapter) Stop()  { a.bot.Stop() }

func (a *BotAdapter) handleStart(c tele.Context) error {
	msg := "<b>UpMedia — ИИ-помощник новостной редакции</b>\n\n" +
		"• Отправьте новость или ссылку для анализа происхождения, проверки фактов и генерации черновиков.\n" +
		"• Используйте <code>/suggest текст</code> или отправьте фото с подписью для тестирования потока пользовательской «предложки»."
	return c.Send(msg)
}

// Обработка команды /suggest (изолированный поток предложки от читателей)
func (a *BotAdapter) handleSuggestionCommand(c tele.Context) error {
	payload := strings.TrimSpace(c.Message().Payload)
	if payload == "" {
		return c.Send("Отправьте новость в формате: <code>/suggest очевидцы сообщают о...</code>")
	}
	return a.processInput(c, payload, true)
}

func (a *BotAdapter) handleIncomingPhoto(c tele.Context) error {
	caption := c.Message().Caption
	if caption == "" {
		caption = "Фотоматериал очевидца без текстового описания"
	}
	return a.processInput(c, caption, true)
}

func (a *BotAdapter) handleIncomingText(c tele.Context) error {
	return a.processInput(c, c.Text(), false)
}

func (a *BotAdapter) processInput(c tele.Context, text string, isSuggestion bool) error {
	ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
	defer cancel()

	statusMsg, err := a.bot.Send(c.Recipient(), "⏳ <i>Кластеризация, поиск первоисточника и проверка утверждений...</i>")
	if err != nil {
		return err
	}

	cluster, err := a.service.ProcessIncoming(ctx, text, isSuggestion)
	if err != nil {
		_, editErr := a.bot.Edit(statusMsg, "❌ Не удалось обработать материал.")
		return errors.Join(err, editErr)
	}

	_, err = a.bot.Edit(
		statusMsg,
		a.presenter.RenderEditorCard(cluster),
		a.presenter.BuildEditorKeyboard(cluster),
		tele.NoPreview,
	)
	return err
}

func (a *BotAdapter) handleSwitchFormat(format domain.PostFormat) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
		defer cancel()

		cluster, err := a.service.SwitchFormat(ctx, c.Callback().Data, format)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Карточка не найдена", ShowAlert: true})
		}

		_ = c.Respond(&tele.CallbackResponse{Text: "Формат изменен"})
		return c.Edit(a.presenter.RenderEditorCard(cluster), a.presenter.BuildEditorKeyboard(cluster), tele.NoPreview)
	}
}

func (a *BotAdapter) handleCycleHeadline(c tele.Context) error {
	ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
	defer cancel()

	cluster, err := a.service.CycleHeadline(ctx, c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка смены заголовка", ShowAlert: true})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: "Выбран следующий вариант заголовка"})
	return c.Edit(a.presenter.RenderEditorCard(cluster), a.presenter.BuildEditorKeyboard(cluster), tele.NoPreview)
}

func (a *BotAdapter) handleRecheck(c tele.Context) error {
	ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
	defer cancel()

	_ = c.Respond(&tele.CallbackResponse{Text: "🔎 Запущена перепроверка по официальным источникам..."})

	cluster, err := a.service.RequestAdditionalCheck(ctx, c.Callback().Data)
	if err != nil {
		return err
	}

	return c.Edit(a.presenter.RenderEditorCard(cluster), a.presenter.BuildEditorKeyboard(cluster), tele.NoPreview)
}

func (a *BotAdapter) handleFeedback(fb domain.FeedbackType) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
		defer cancel()

		cluster, err := a.service.RecordFeedback(ctx, c.Callback().Data, fb)
		if err != nil {
			return err
		}

		if fb == domain.FeedbackUninteresting {
			_ = c.Respond(&tele.CallbackResponse{Text: "👎 Профиль рекомендаций обновлен. Карточка скрыта."})
			return c.Delete()
		}

		_ = c.Respond(&tele.CallbackResponse{Text: "👍 Учтено в персональных рекомендациях канала!"})
		return c.Edit(a.presenter.RenderEditorCard(cluster), a.presenter.BuildEditorKeyboard(cluster), tele.NoPreview)
	}
}

func (a *BotAdapter) handlePublish(c tele.Context) error {
	ctx, cancel := context.WithTimeout(a.baseCtx, a.handlerTimeout)
	defer cancel()

	id := c.Callback().Data
	cluster, err := a.service.GetCluster(ctx, id)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Карточка не найдена", ShowAlert: true})
	}

	postHTML := a.presenter.RenderChannelPost(cluster)
	published, err := a.service.Publish(ctx, id, postHTML)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyPublished) {
			return c.Respond(&tele.CallbackResponse{Text: "Уже опубликовано!", ShowAlert: true})
		}
		return c.Respond(&tele.CallbackResponse{
			Text:      "Ошибка отправки в канал. Проверьте права бота.",
			ShowAlert: true,
		})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: "🚀 Материал опубликован!"})
	return c.Edit(a.presenter.RenderEditorCard(published), a.presenter.BuildEditorKeyboard(published), tele.NoPreview)
}