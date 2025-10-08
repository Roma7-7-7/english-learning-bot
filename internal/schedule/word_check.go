package schedule

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gopkg.in/telebot.v3"
)

const (
	publishTimeout = 1 * time.Minute
)

type (
	WordCheckConfig struct {
		ChatIDs  []int64
		Interval time.Duration
		HourFrom int
		HourTo   int
		Location *time.Location
	}

	Publisher interface {
		SendWordCheck(ctx context.Context, chatID int64) error
	}
)

func StartWordCheckSchedule(ctx context.Context, conf WordCheckConfig, p Publisher, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "panic", "error", r)
		}
	}()

	log.InfoContext(ctx, "word check schedule started")
	defer log.InfoContext(ctx, "word check schedule stopped")
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				log.DebugContext(ctx, "word check schedule stopped")
			} else {
				log.ErrorContext(ctx, "word check schedule stopped", "error", ctx.Err())
			}
		case <-time.After(conf.Interval):
			log.DebugContext(ctx, "word check execution started")
			now := time.Now().In(conf.Location)
			if now.Hour() < conf.HourFrom || now.Hour() >= conf.HourTo {
				log.DebugContext(ctx, "word check execution skipped", "current_hour", now.Hour())
				continue
			}

			for _, chatID := range conf.ChatIDs {
				ctx, cancel := context.WithTimeout(ctx, publishTimeout)
				log.DebugContext(ctx, "sending word check", "chat_id", chatID)
				if err := p.SendWordCheck(ctx, chatID); err != nil {
					if errors.Is(err, telebot.ErrBlockedByUser) {
						log.InfoContext(ctx, "user blocked bot", "chat_id", chatID)
						continue
					}
					log.ErrorContext(ctx, "failed to send word check", "error", err, "chat_id", chatID)
				}
				cancel()
			}
		}
	}
}
