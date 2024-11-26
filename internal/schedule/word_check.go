package schedule

import (
	"context"
	"log/slog"
	"time"
)

const (
	publishTimeout = 1 * time.Minute
)

var kyivTime *time.Location

type Publisher interface {
	SendWordCheck(ctx context.Context, chatID int64) error
}

func StartWordCheckSchedule(ctx context.Context, chatIDS []int64, interval time.Duration, p Publisher, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic", "error", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			log.Info("word check schedule started")
			if time.Now().In(kyivTime).Hour() < 9 || time.Now().In(kyivTime).Hour() > 23 {
				continue
			}

			for _, chatID := range chatIDS {
				ctx, cancel := context.WithTimeout(ctx, publishTimeout)
				if err := p.SendWordCheck(ctx, chatID); err != nil {
					log.Error("failed to send word check", "error", err, "chat_id", chatID)
				}
				cancel()
			}
			log.Info("word check schedule finished")
		}
	}
}

func init() {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic(err)
	}
	kyivTime = loc
	slog.Info("initialized kyiv time location", "current_time", time.Now().In(kyivTime).Format(time.RFC3339))
}
