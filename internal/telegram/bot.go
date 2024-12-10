package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"text/template"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

const (
	commandStart    = "/start"
	commandAdd      = "/add"
	commandStats    = "/stats"
	commandToReview = "/to_review"
	commandRandom   = "/random"
	commandMute     = "/mute"

	callbackSeeTranslation = "callback#see_translation"
	callbackResetToReview  = "callback#reset_to_review"
	callbackWordGuessed    = "callback#word#guessed"
	callbackWordMissed     = "callback#word#missed"
	callbackWordToReview   = "callback#word#to_review"

	somethingWentWrongMsg = "something went wrong"

	processTimeout = 10 * time.Second

	muteDuration               = 1 * time.Hour
	callbackDataExpirationTime = 24 * 7 * time.Hour
)

//nolint:gochecknoglobals // it's a template for rendering words to review
var wordsToReviewTemplate = template.Must(template.New("to_review").
	Parse(`To Review:
{{- range .}}
- {{.Word}}
{{- end}}
`))

type (
	Bot struct {
		bot  *tb.Bot
		repo dal.Repository

		middlewares []tb.MiddlewareFunc

		silentMode *sync.Map
		log        *slog.Logger
	}

	replier interface {
		Reply(any, ...any) error
	}

	noOpReplier struct{}
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
		middlewares: middlewares,
		silentMode:  &sync.Map{},
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
	b.bot.Handle(commandMute, b.HandleMute, b.middlewares...)

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
		b.log.ErrorContext(ctx, "failed to get stats", "error", err)
		return m.Reply("failed to get stats")
	}

	return m.Reply(fmt.Sprintf("15+: %d\n10-14: %d\n1-9: %d\nTotal: %d", stats.GreaterThanOrEqual15, stats.Between10And14, stats.Between1And9, stats.Total))
}

func (b *Bot) HandleAdd(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	parts := strings.Split(strings.ToLower(m.Text())[len(commandAdd):], ":")
	if len(parts) != 2 { //nolint: mnd // word:translation
		b.log.DebugContext(ctx, "wrong message format", "message", m.Message().Text)
		return m.Reply("wrong message format, it should be like: word:translation")
	}

	word, translation := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if err := b.repo.AddWordTranslation(ctx, m.Chat().ID, word, translation); err != nil {
		b.log.ErrorContext(ctx, "failed to add translation", "error", err)
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
		b.log.ErrorContext(ctx, "failed to get words to review", "error", err)
		return m.Reply("failed to get words to review")
	}

	if len(words) == 0 {
		return m.Reply("no words to review")
	}

	buff := &strings.Builder{}
	if err = wordsToReviewTemplate.Execute(buff, words); err != nil {
		b.log.ErrorContext(ctx, "failed to render words to review", "error", err)
		return m.Reply("failed to render words to review")
	}

	return m.Reply(buff.String(), wordsToReviewMarkup())
}

func (b *Bot) SendWordCheck(ctx context.Context, chatID int64) error {
	return b.sendWordCheck(ctx, chatID, &noOpReplier{})
}

func (b *Bot) sendWordCheck(ctx context.Context, chatID int64, replier replier) error {
	wt, err := b.repo.FindRandomBatchedWordTranslation(ctx, chatID)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.DebugContext(ctx, "no words to check", "chatID", chatID)
			return replier.Reply("no words to check")
		}

		b.log.ErrorContext(ctx, "failed to get random translation", "error", err)
		return replier.Reply(somethingWentWrongMsg)
	}

	data := dal.CallbackData{
		ChatID:    chatID,
		Word:      wt.Word,
		ExpiresAt: time.Now().Add(callbackDataExpirationTime),
	}
	callbackID, err := b.repo.InsertCallback(ctx, data)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to insert callback data", "error", err)
		return replier.Reply(somethingWentWrongMsg)
	}

	opts := make([]any, 0, 3) //nolint: mnd // 2 by default and 1 more for optional silent mode
	opts = append(opts, tb.ModeMarkdownV2, seeTranslationMarkup(callbackID))
	if v, ok := b.silentMode.Load(chatID); ok {
		if until, okt := v.(time.Time); okt && time.Now().Before(until) {
			opts = append(opts, tb.Silent)
		}
	}

	_, err = b.bot.Send(tb.ChatID(chatID), normalizeMessage(fmt.Sprintf("**%s**", wt.Word)), opts...)
	return err
}

func (b *Bot) HandleCallback(c tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	data := c.Callback().Data
	parts := strings.Split(data, ":")

	if len(parts) > 2 { //nolint: mnd // key:<cacheUUID>
		b.log.Warn("wrong callback data", "data", data)
		return c.RespondText(somethingWentWrongMsg)
	}

	if parts[0] == callbackResetToReview {
		if err := b.repo.ResetToReview(ctx, c.Chat().ID); err != nil {
			b.log.ErrorContext(ctx, "failed to reset to review", "error", err)
			return c.RespondText(somethingWentWrongMsg)
		}

		return c.Delete()
	}

	cData, err := b.repo.FindCallback(ctx, c.Chat().ID, parts[1])
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.Warn("callback data not found", "data", data)
			return c.RespondText("too much time passed")
		}

		b.log.ErrorContext(ctx, "failed to find callback data", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}

	switch parts[0] {
	case callbackSeeTranslation:
		var wt *dal.WordTranslation
		wt, err = b.repo.FindWordTranslation(ctx, c.Chat().ID, cData.Word)
		if err != nil {
			b.log.ErrorContext(ctx, "failed to get word translation", "error", err)
			return c.RespondText(somethingWentWrongMsg)
		}
		msg := fmt.Sprintf("**%s**", wt.Translation)
		if wt.Description != "" {
			msg += fmt.Sprintf(": _%s_", wt.Description)
		}
		err = c.Send(normalizeMessage(msg), tb.ModeMarkdownV2, guessedResponseMarkup(cData.ID))
	case callbackWordGuessed:
		err = b.repo.IncreaseGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordMissed:
		err = b.repo.ResetGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordToReview:
		err = b.repo.MarkToReviewAndResetStreak(ctx, c.Chat().ID, cData.Word)
	default:
		b.log.Warn("unknown callback action", "action", parts[0])
		return c.RespondText(somethingWentWrongMsg)
	}

	if err != nil {
		b.log.ErrorContext(ctx, "failed to process callback", "error", err)
		return c.RespondText(somethingWentWrongMsg)
	}

	return c.Delete()
}

func (b *Bot) HandleMute(m tb.Context) error {
	chatID := m.Chat().ID
	b.silentMode.Store(chatID, time.Now().Add(muteDuration))
	return m.Reply(fmt.Sprintf("muted for %s", muteDuration), tb.Silent)
}

func (r *noOpReplier) Reply(any, ...any) error {
	return nil
}

func seeTranslationMarkup(uuid string) *tb.ReplyMarkup {
	return &tb.ReplyMarkup{
		InlineKeyboard: [][]tb.InlineButton{
			{
				{
					Text: "See translation",
					Data: fmt.Sprintf("%s:%s", callbackSeeTranslation, uuid),
				},
			},
		},
	}
}

func guessedResponseMarkup(uuid string) *tb.ReplyMarkup {
	return &tb.ReplyMarkup{
		InlineKeyboard: [][]tb.InlineButton{
			{
				{
					Text: "[      ✅      ]",
					Data: fmt.Sprintf("%s:%s", callbackWordGuessed, uuid),
				},
				{
					Text: "[      ❌      ]",
					Data: fmt.Sprintf("%s:%s", callbackWordMissed, uuid),
				},
				{
					Text: "[      ❓      ]",
					Data: fmt.Sprintf("%s:%s", callbackWordToReview, uuid),
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

func processCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), processTimeout)
}

//nolint:gochecknoglobals // it's a list of characters to escape
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
