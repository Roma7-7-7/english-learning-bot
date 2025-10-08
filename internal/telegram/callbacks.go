package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	tb "gopkg.in/telebot.v3"
)

type callbackData struct {
	Action           string
	TargetIdentifier string
}

func (b *Bot) HandleCallback(c tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	data, err := parseCallbackData(c.Callback().Data)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to parse callback data", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}

	switch data.Action {
	case callbackAuthConfirm:
		return b.handleAuthConfirmCallback(ctx, c, data)
	case callbackAuthDecline:
		return b.handleAuthDeclineCallback(ctx, c, data)
	}

	cData, err := b.repo.FindCallback(ctx, c.Chat().ID, data.TargetIdentifier)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.Warn("callback data not found", "data", data)
			return c.RespondText("too much time passed")
		}

		b.log.ErrorContext(ctx, "failed to find callback data", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}

	switch data.Action {
	case callbackSeeTranslation:
		err = b.handleSeeTranslationCallback(ctx, c, cData)
	case callbackWordGuessed:
		err = b.handleWordGuessedCallback(ctx, c, cData)
	case callbackWordMissed:
		err = b.handleWordMissedCallback(ctx, c, cData)
	case callbackWordToReview:
		err = b.handleWordToReviewCallback(ctx, c, cData)
	default:
		b.log.Warn("unknown callback action", "action", data.Action)
		return c.RespondText(somethingWentWrongMsg)
	}

	if err != nil {
		b.log.ErrorContext(ctx, "failed to process callback", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}

	return c.Delete()
}

func (b *Bot) handleAuthConfirmCallback(ctx context.Context, c tb.Context, data callbackData) error {
	if err := b.repo.ConfirmAuthConfirmation(ctx, c.Chat().ID, data.TargetIdentifier); err != nil {
		b.log.ErrorContext(ctx, "failed to confirm callback data", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}
	return c.Delete()
}

func (b *Bot) handleAuthDeclineCallback(ctx context.Context, c tb.Context, data callbackData) error {
	if err := b.repo.DeleteAuthConfirmation(ctx, c.Chat().ID, data.TargetIdentifier); err != nil {
		b.log.ErrorContext(ctx, "failed to decline callback data", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}
	return c.Delete()
}

func (b *Bot) handleSeeTranslationCallback(ctx context.Context, c tb.Context, data *dal.CallbackData) error {
	wt, err := b.repo.FindWordTranslation(ctx, c.Chat().ID, data.Word)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to get word translation", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}
	msg := fmt.Sprintf("**%s**", wt.Translation)
	if wt.Description != "" {
		msg += fmt.Sprintf(": _%s_", wt.Description)
	}
	return c.Send(normalizeMessage(msg), guessedResponseMarkup(data.ID), tb.ModeMarkdownV2, tb.Silent)
}

func (b *Bot) handleWordGuessedCallback(ctx context.Context, c tb.Context, data *dal.CallbackData) error {
	return b.repo.Transact(ctx, func(r dal.Repository) error {
		if err := r.IncreaseGuessedStreak(ctx, c.Chat().ID, data.Word); err != nil {
			return fmt.Errorf("increase guessed streak: %w", err)
		}
		if err := r.IncrementWordGuessed(ctx, c.Chat().ID); err != nil {
			return fmt.Errorf("increment word guessed: %w", err)
		}
		if err := r.UpdateTotalWordsLearned(ctx, c.Chat().ID); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

func (b *Bot) handleWordMissedCallback(ctx context.Context, c tb.Context, cData *dal.CallbackData) error {
	return b.repo.Transact(ctx, func(r dal.Repository) error {
		if err := r.ResetGuessedStreak(ctx, c.Chat().ID, cData.Word); err != nil {
			return fmt.Errorf("reset guessed streak: %w", err)
		}
		if err := r.IncrementWordMissed(ctx, c.Chat().ID); err != nil {
			return fmt.Errorf("increment word missed: %w", err)
		}
		if err := r.UpdateTotalWordsLearned(ctx, c.Chat().ID); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

func (b *Bot) handleWordToReviewCallback(ctx context.Context, c tb.Context, cData *dal.CallbackData) error {
	return b.repo.Transact(ctx, func(r dal.Repository) error {
		if err := r.MarkToReview(ctx, c.Chat().ID, cData.Word, true); err != nil {
			return fmt.Errorf("mark to review: %w", err)
		}
		return nil
	})
}

func parseCallbackData(val string) (callbackData, error) {
	val = strings.TrimSpace(val)
	parts := strings.Split(val, ":")
	if len(parts) != 2 { //nolint:mnd // it's ok
		return callbackData{}, fmt.Errorf("invalid callback data: %s", val)
	}
	return callbackData{
		Action:           parts[0],
		TargetIdentifier: parts[1],
	}, nil
}
