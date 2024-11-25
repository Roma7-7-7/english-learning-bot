package telegram

import (
	"fmt"
	"log/slog"

	tb "gopkg.in/telebot.v3"
)

func Recover(log *slog.Logger) tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) error {
			defer func() {
				if r := recover(); r != nil {
					log.Error("panic occurred", "panic", r)
				}
			}()
			return next(c)
		}
	}
}

func LogErrors(log *slog.Logger) tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) error {
			err := next(c)
			if err != nil {
				log.Error("failed to process message", "error", err)
			}
			return err
		}
	}
}

func AllowedChats(ids []int64) tb.MiddlewareFunc {
	idsMap := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		idsMap[id] = struct{}{}
	}
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) error {
			chatID := c.Chat().ID
			if _, ok := idsMap[chatID]; !ok {
				return fmt.Errorf("chat %d is not allowed", chatID)
			}

			return next(c)
		}
	}
}
