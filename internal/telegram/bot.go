package telegram

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/pkg/cache"
	tb "gopkg.in/telebot.v3"
)

const (
	commandStart  = "/start"
	commandAdd    = "/add"
	commandRandom = "/random"

	callbackSeeTranslation = "callback#see_translation"
	callbackWordGuessed    = "callback#word#guessed"
	callbackWordMissed     = "callback#word#missed"

	cacheTTL       = 24 * time.Hour
	processTimeout = 10 * time.Second
)

type (
	Cache interface {
		Get(key string) (string, bool)
		Set(key, value string, ttl time.Duration)
	}

	Bot struct {
		bot   *tb.Bot
		repo  dal.Repository
		cache Cache

		middlewares []tb.MiddlewareFunc

		log *slog.Logger
	}
)

func NewBot(token string, repo dal.Repository, log *slog.Logger, middlewares ...tb.MiddlewareFunc) (*Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token: token,
		Poller: &tb.LongPoller{
			Timeout: 1 * time.Minute,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	return &Bot{
		bot:         b,
		repo:        repo,
		cache:       cache.NewInMemory(),
		middlewares: middlewares,
		log:         log,
	}, nil
}

func (b *Bot) Start() {
	b.bot.Handle(commandStart, b.HandleStart, b.middlewares...)
	b.bot.Handle(commandAdd, b.HandleAdd, b.middlewares...)
	b.bot.Handle(commandRandom, b.HandleRandom, b.middlewares...)
	b.bot.Handle(tb.OnCallback, b.HandleCallback, b.middlewares...)

	b.bot.Start()
}

func (b *Bot) HandleStart(m tb.Context) error {
	return m.Reply("Hello, I'm a translation bot. To add a translation use /add command. Example: /add word: translation")
}

func (b *Bot) HandleAdd(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	parts := strings.Split(strings.ToLower(m.Text())[len(commandAdd):], ":")
	if len(parts) != 2 {
		slog.Debug("wrong message format", "message", m.Message().Text)
		return m.Reply("wrong message format, it should be like: word:translation")
	}

	word, translation := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if err := b.repo.AddWordTranslation(ctx, m.Chat().ID, word, translation); err != nil {
		slog.Error("failed to add translation", "error", err)
		return m.Reply("failed to add translation")
	}

	return m.Reply("translation added")
}

func (b *Bot) HandleRandom(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	return b.sendWordCheck(ctx, m.Chat().ID, m)
}

func (b *Bot) SendWordCheck(ctx context.Context, chatID int64) error {
	return b.sendWordCheck(ctx, chatID, nil)
}

func (b *Bot) sendWordCheck(ctx context.Context, chatID int64, m tb.Context) error {
	wt, err := b.repo.GetRandomBatchedWordTranslation(ctx, chatID)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.Debug("no words to check", "chatID", chatID)
			if m != nil {
				return m.Reply("no words to check")
			}

			return nil
		}

		if m != nil {
			b.log.Error("failed to get random translation", "error", err)
			return m.Reply("failed to get random translation")
		}

		return fmt.Errorf("get random translation: %w", err)
	}

	cacheID := callbackCacheID(chatID)
	b.cache.Set(cacheID, encodeBase64(wt.Word)+":"+encodeBase64(wt.Translation)+":"+encodeBase64(wt.Description), cacheTTL)

	_, err = b.bot.Send(tb.ChatID(chatID), normalizeMessage(fmt.Sprintf("**%s**", wt.Word)), tb.ModeMarkdownV2, seeTranslationMarkup(cacheID))
	return err
}

func (b *Bot) HandleCallback(c tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	data := c.Callback().Data
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		slog.Warn("wrong callback data", "data", data)
		return c.RespondText("something went wrong")
	}

	cacheID := parts[1]
	cached, ok := b.cache.Get(cacheID)
	if !ok {
		slog.Debug("cached data not found", "cacheID", cacheID)
		return c.RespondText("too much time passed, please try new random word")
	}
	cachedParts := strings.Split(cached, ":")
	if len(cachedParts) > 3 {
		slog.Warn("wrong cached data", "data", cached)
		return c.RespondText("something went wrong")
	}

	word, err := decodeBase64(cachedParts[0])
	if err != nil {
		slog.Warn("failed to decode word", "word", cachedParts[0], "error", err)
		return c.RespondText("something went wrong")
	}
	translation, err := decodeBase64(cachedParts[1])
	if err != nil {
		slog.Warn("failed to decode translation", "translation", cachedParts[1], "error", err)
		return c.RespondText("something went wrong")
	}
	description := ""
	if len(cachedParts) > 2 {
		description, err = decodeBase64(cachedParts[2])
		if err != nil {
			slog.Warn("failed to decode description", "description", cachedParts[2], "error", err)
			return c.RespondText("something went wrong")
		}
	}

	switch parts[0] {
	case callbackSeeTranslation:
		msg := fmt.Sprintf("**%s**", translation)
		if description != "" {
			msg += fmt.Sprintf(": _%s_", description)
		}
		err = c.Send(normalizeMessage(msg), tb.ModeMarkdownV2, guessedResponseMarkup(cacheID))
	case callbackWordGuessed:
		err = b.repo.IncreaseGuessedStreak(ctx, c.Chat().ID, word)
	case callbackWordMissed:
		err = b.repo.ResetGuessedStreak(ctx, c.Chat().ID, word)
	default:
		slog.Warn("unknown callback action", "action", parts[0])
		return c.RespondText("something went wrong")
	}

	if err != nil {
		slog.Error("failed to process callback", "error", err)
		return c.RespondText("something went wrong")
	}

	return c.Delete()
}

func callbackCacheID(chatID int64) string {
	return fmt.Sprintf("%d#%d", chatID, time.Now().UnixNano())
}

func seeTranslationMarkup(cacheID string) *tb.ReplyMarkup {
	return &tb.ReplyMarkup{
		InlineKeyboard: [][]tb.InlineButton{
			{
				{
					Text: "See translation",
					Data: fmt.Sprintf("%s:%s", callbackSeeTranslation, cacheID),
				},
			},
		},
	}
}

func guessedResponseMarkup(cacheID string) *tb.ReplyMarkup {
	return &tb.ReplyMarkup{
		InlineKeyboard: [][]tb.InlineButton{
			{
				{
					Text: "✅",
					Data: fmt.Sprintf("%s:%s", callbackWordGuessed, cacheID),
				},
				{
					Text: "❌",
					Data: fmt.Sprintf("%s:%s", callbackWordMissed, cacheID),
				},
			},
		},
	}
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func decodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func processCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), processTimeout)
}

var toEscape = []string{
	"=",
	"-",
	")",
}

func normalizeMessage(s string) string {
	res := strings.TrimSpace(strings.ToLower(s))
	for _, esc := range toEscape {
		res = strings.ReplaceAll(res, esc, "\\"+esc)
	}
	return res
}
