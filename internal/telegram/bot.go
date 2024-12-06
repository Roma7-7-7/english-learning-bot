package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/pkg/cache"
	tb "gopkg.in/telebot.v3"
)

const (
	commandStart    = "/start"
	commandAdd      = "/add"
	commandStats    = "/stats"
	commandToReview = "/to_review"
	commandRandom   = "/random"

	callbackSeeTranslation = "callback#see_translation"
	callbackResetToReview  = "callback#reset_to_review"
	callbackWordGuessed    = "callback#word#guessed"
	callbackWordMissed     = "callback#word#missed"
	callbackWordToReview   = "callback#word#to_review"

	somethingWentWrongMsg = "something went wrong"

	cacheTTL       = 24 * time.Hour
	processTimeout = 10 * time.Second
)

var wordsToReviewTemplate = template.Must(template.New("to_review").
	Parse(`To Review:
{{- range .}}
- {{.Word}}
{{- end}}
`))

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

	callbackData struct {
		CacheID     string `json:"cache_id"`
		Word        string `json:"word"`
		Translation string `json:"translation"`
		Description string `json:"description"`
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
	b.bot.Handle(commandStats, b.HandleStats, b.middlewares...)
	b.bot.Handle(commandToReview, b.HandleToReview, b.middlewares...)
	b.bot.Handle(commandRandom, b.HandleRandom, b.middlewares...)
	b.bot.Handle(tb.OnCallback, b.HandleCallback, b.middlewares...)

	b.bot.Start()
}

func (b *Bot) HandleStart(m tb.Context) error {
	return m.Reply("Hello, I'm a translation bot. To add a translation use /add command. Example: /add word: translation")
}

func (b *Bot) HandleStats(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	stats, err := b.repo.GetStats(ctx, m.Chat().ID)
	if err != nil {
		slog.Error("failed to get stats", "error", err)
		return m.Reply("failed to get stats")
	}

	return m.Reply(fmt.Sprintf("15+: %d\n10-14: %d\n1-9: %d\nTotal: %d", stats.GreaterThanOrEqual15, stats.Between10And14, stats.Between1And9, stats.Total))
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

func (b *Bot) HandleToReview(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	words, err := b.repo.FindWordsToReview(ctx, m.Chat().ID)
	if err != nil {
		slog.Error("failed to get words to review", "error", err)
		return m.Reply("failed to get words to review")
	}

	if len(words) == 0 {
		return m.Reply("no words to review")
	}

	buff := &strings.Builder{}
	if err := wordsToReviewTemplate.Execute(buff, words); err != nil {
		slog.Error("failed to render words to review", "error", err)
		return m.Reply("failed to render words to review")
	}

	return m.Reply(buff.String(), wordsToReviewMarkup())
}

func (b *Bot) SendWordCheck(ctx context.Context, chatID int64) error {
	return b.sendWordCheck(ctx, chatID, nil)
}

func (b *Bot) sendWordCheck(ctx context.Context, chatID int64, m tb.Context) error {
	wt, err := b.repo.FindRandomBatchedWordTranslation(ctx, chatID)
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
	data, err := encodeCallbackData(callbackData{
		CacheID:     cacheID,
		Word:        wt.Word,
		Translation: wt.Translation,
		Description: wt.Description,
	})
	if err != nil {
		b.log.Error("failed to marshal callback data", "error", err)
		return m.Reply(somethingWentWrongMsg)
	}
	b.cache.Set(cacheID, data, cacheTTL)

	_, err = b.bot.Send(tb.ChatID(chatID), normalizeMessage(fmt.Sprintf("**%s**", wt.Word)), tb.ModeMarkdownV2, seeTranslationMarkup(cacheID))
	return err
}

func (b *Bot) HandleCallback(c tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	data := c.Callback().Data
	parts := strings.Split(data, ":")
	if len(parts) > 2 {
		slog.Warn("wrong callback data", "data", data)
		return c.RespondText("something went wrong")
	}

	var cData callbackData
	var err error
	if len(parts) == 2 {
		cached, ok := b.cache.Get(parts[1])
		if !ok {
			slog.Warn("callback data not found", "data", data)
			return c.RespondText("something went wrong")
		}

		cData, err = decodeCallbackData(cached)
		if err != nil {
			slog.Error("failed to decode callback data", "error", err)
			return c.RespondText(somethingWentWrongMsg)
		}
	}

	switch parts[0] {
	case callbackSeeTranslation:
		msg := fmt.Sprintf("**%s**", cData.Translation)
		if cData.Description != "" {
			msg += fmt.Sprintf(": _%s_", cData.Description)
		}
		err = c.Send(normalizeMessage(msg), tb.ModeMarkdownV2, guessedResponseMarkup(cData.CacheID))
	case callbackResetToReview:
		err = b.repo.ResetToReview(ctx, c.Chat().ID)
	case callbackWordGuessed:
		err = b.repo.IncreaseGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordMissed:
		err = b.repo.ResetGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordToReview:
		err = b.repo.MarkToReviewAndResetStreak(ctx, c.Chat().ID, cData.Word)
	default:
		slog.Warn("unknown callback action", "action", parts[0])
		return c.RespondText(somethingWentWrongMsg)
	}

	if err != nil {
		slog.Error("failed to process callback", "error", err)
		return c.RespondText(somethingWentWrongMsg)
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
				{
					Text: "❓",
					Data: fmt.Sprintf("%s:%s", callbackWordToReview, cacheID),
				},
			},
		},
	}
}

func wordsToReviewMarkup() *tb.ReplyMarkup {
	return &tb.ReplyMarkup{
		InlineKeyboard: [][]tb.InlineButton{
			{
				{
					Text: "✅",
					Data: callbackResetToReview,
				},
			},
		},
	}
}

func encodeCallbackData(data callbackData) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal callback data: %w", err)
	}
	return string(b), nil
}

func decodeCallbackData(data string) (callbackData, error) {
	var cData callbackData
	if err := json.Unmarshal([]byte(data), &cData); err != nil {
		return callbackData{}, fmt.Errorf("unmarshal callback data: %w", err)
	}
	return cData, nil
}

func processCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), processTimeout)
}

var toEscape = []string{
	"=",
	"-",
	"(",
	")",
}

func normalizeMessage(s string) string {
	res := strings.TrimSpace(strings.ToLower(s))
	for _, esc := range toEscape {
		res = strings.ReplaceAll(res, esc, "\\"+esc)
	}
	return res
}
