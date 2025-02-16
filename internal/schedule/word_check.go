package schedule

import (
	"context"
	"log/slog"
	"time"
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
			return
		case <-time.After(conf.Interval):
			log.DebugContext(ctx, "word check execution started")
			now := time.Now().In(conf.Location)
			if now.Hour() < conf.HourFrom || now.Hour() >= conf.HourTo {
				log.DebugContext(ctx, "word check execution skipped", "current_hour", now.Hour())
				continue
			}

			for _, chatID := range conf.ChatIDs {
				ctx, cancel := context.WithTimeout(ctx, publishTimeout) //nolint:govet // it is supposed to override ctx here
				log.DebugContext(ctx, "sending word check", "chat_id", chatID)
				if err := p.SendWordCheck(ctx, chatID); err != nil {
					log.ErrorContext(ctx, "failed to send word check", "error", err, "chat_id", chatID)
				}
				cancel()
			}
		}
	}
}
