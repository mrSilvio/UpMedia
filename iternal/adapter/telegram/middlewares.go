package telegram

import (
	"log/slog"
	"runtime/debug"
	"time"

	tele "gopkg.in/telebot.v3"
)

// RecoverMiddleware защищает процесс от падения при panic внутри хендлеров
func RecoverMiddleware(logger *slog.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered in telegram handler",
						slog.Any("panic", r),
						slog.String("stack", string(debug.Stack())),
					)
					_ = c.Send("⚠️ Внутренняя ошибка сервера. Запрос зафиксирован.")
				}
			}()
			return next(c)
		}
	}
}

// SlogMiddleware логирует каждый входящий апдейт и время его обработки
func SlogMiddleware(logger *slog.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			start := time.Now()
			var userID int64
			if c.Sender() != nil {
				userID = c.Sender().ID
			}

			err := next(c)

			logger.Info("telegram update processed",
				slog.Int("update_id", c.Update().ID),
				slog.Int64("user_id", userID),
				slog.Duration("latency", time.Since(start)),
				slog.Any("error", err),
			)
			return err
		}
	}
}

// EditorAuthMiddleware пускает только разрешенных редакторов (если список задан)
func EditorAuthMiddleware(allowedIDs map[int64]struct{}) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if len(allowedIDs) == 0 {
				return next(c) // В демо-режиме на хакатоне можно оставить пустым (доступ всем)
			}
			if c.Sender() == nil {
				return nil
			}
			if _, ok := allowedIDs[c.Sender().ID]; !ok {
				return c.Send("⛔ Доступ разрешен только редакторам UpMedia.")
			}
			return next(c)
		}
	}
}