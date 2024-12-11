package schedule

import (
	"context"
	"log/slog"
	"time"
)

const (
	publishTimeout = 1 * time.Minute
)

type Publisher interface {
	SendWordCheck(ctx context.Context, chatID int64) error
}

func StartWordCheckSchedule(ctx context.Context, chatIDs []int64, interval time.Duration, p Publisher, loc *time.Location, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "panic", "error", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			log.InfoContext(ctx, "word check schedule started")
			if time.Now().In(loc).Hour() < 9 || time.Now().In(loc).Hour() > 23 {
				continue
			}

			for _, chatID := range chatIDs {
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
