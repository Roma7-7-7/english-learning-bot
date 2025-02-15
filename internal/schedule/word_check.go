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

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(conf.Interval):
			log.InfoContext(ctx, "word check schedule started")
			if time.Now().In(conf.Location).Hour() < conf.HourFrom || time.Now().In(conf.Location).Hour() > conf.HourTo {
				continue
			}

			for _, chatID := range conf.ChatIDs {
				ctx, cancel := context.WithTimeout(ctx, publishTimeout) //nolint:govet // it is supposed to override ctx here
				if err := p.SendWordCheck(ctx, chatID); err != nil {
					log.ErrorContext(ctx, "failed to send word check", "error", err, "chat_id", chatID)
				}
				cancel()
			}
			log.InfoContext(ctx, "word check schedule finished")
		}
	}
}
