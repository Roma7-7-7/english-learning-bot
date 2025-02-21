package telegram

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strings"
	"sync"
	"text/template"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/data"
)

const (
	commandStart    = "/start"
	commandGet      = "/get"
	commandAdd      = "/add"
	commandUpdate   = "/update"
	commandDelete   = "/delete"
	commandStats    = "/stats"
	commandToReview = "/to_review"
	commandRandom   = "/random"

	callbackAuthConfirm    = "callback#auth#confirm"
	callbackAuthDecline    = "callback#auth#decline"
	callbackSeeTranslation = "callback#see_translation"
	callbackResetToReview  = "callback#reset_to_review"
	callbackWordGuessed    = "callback#word#guessed"
	callbackWordMissed     = "callback#word#missed"
	callbackWordToReview   = "callback#word#to_review"

	somethingWentWrongMsg = "something went wrong"

	processTimeout = 10 * time.Second

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

		log *slog.Logger
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
		log:         log,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	b.bot.Handle(commandStart, b.HandleStart, b.middlewares...)
	b.bot.Handle(commandGet, b.HandleGet, b.middlewares...)
	b.bot.Handle(commandAdd, b.HandleAdd, b.middlewares...)
	b.bot.Handle(commandUpdate, b.HandleUpdate, b.middlewares...)
	b.bot.Handle(commandDelete, b.HandleDelete, b.middlewares...)
	b.bot.Handle(commandStats, b.HandleStats, b.middlewares...)
	b.bot.Handle(commandToReview, b.HandleToReview, b.middlewares...)
	b.bot.Handle(commandRandom, b.HandleRandom, b.middlewares...)
	b.bot.Handle(tb.OnCallback, b.HandleCallback, b.middlewares...)
	b.bot.Handle(tb.OnDocument, b.HandleDocument, b.middlewares...)

	go func() {
		time.Sleep(5 * time.Second) //nolint:mnd // wait for the bot to start
		<-ctx.Done()
		b.log.InfoContext(ctx, "stopping telebot instance")
		b.bot.Stop()
		b.log.InfoContext(ctx, "telebot instance stopped")
	}()

	b.log.InfoContext(ctx, "starting telebot instance")
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

func (b *Bot) HandleGet(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	word := strings.TrimSpace(m.Text()[len(commandGet):])

	wt, err := b.repo.FindWordTranslation(ctx, m.Chat().ID, word)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.DebugContext(ctx, "word not found", "word", word)
			return m.Reply("word not found")
		}

		b.log.ErrorContext(ctx, "failed to get word translation", "error", err)
		return m.Reply("failed to get word translation")
	}

	return m.Reply(fmt.Sprintf("**%s**: %s", wt.Word, wt.Translation), tb.ModeMarkdownV2)
}

func (b *Bot) HandleAdd(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	parts := strings.Split(strings.ToLower(m.Text())[len(commandAdd):], ":")
	if len(parts) != 2 { //nolint: mnd // word:translation
		b.log.DebugContext(ctx, "wrong message format", "message", m.Message().Text)
		return m.Reply("wrong message format, it should be like \"word:translation\"")
	}

	word, translation := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if err := b.repo.AddWordTranslation(ctx, m.Chat().ID, word, translation, ""); err != nil {
		b.log.ErrorContext(ctx, "failed to add translation", "error", err)
		return m.Reply("failed to add translation")
	}

	return m.Reply("translation added")
}

func (b *Bot) HandleUpdate(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	parts := strings.Split(strings.ToLower(m.Text())[len(commandUpdate):], ":")
	if len(parts) != 3 { //nolint: mnd // word:translation
		b.log.DebugContext(ctx, "wrong message format", "message", m.Message().Text)
		return m.Reply("wrong message format, it should be like: \"original word:new word:new translation\"")
	}

	original, updated, translation := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])

	if err := b.repo.UpdateWordTranslation(ctx, m.Chat().ID, original, updated, translation, ""); err != nil {
		b.log.ErrorContext(ctx, "failed to update translation", "error", err)
		return m.Reply("failed to update translation")
	}

	return m.Reply("translation updated")
}

func (b *Bot) HandleDelete(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	word := strings.TrimSpace(m.Text()[len(commandDelete):])

	if err := b.repo.DeleteWordTranslation(ctx, m.Chat().ID, word); err != nil {
		b.log.ErrorContext(ctx, "failed to delete translation", "error", err)
		return m.Reply("failed to delete translation")
	}

	return m.Reply("translation deleted")
}

func (b *Bot) HandleRandom(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	return b.sendWordCheck(ctx, m.Chat().ID, dal.FindRandomWordFilter{StreakLimitDirection: dal.LimitDirectionGreaterThanOrEqual, StreakLimit: 0}, m)
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
	filter := dal.FindRandomWordFilter{Batched: true}

	rnd, err := rand.Int(rand.Reader, big.NewInt(100)) //nolint:mnd // 100 is a magic number
	if err != nil {
		b.log.ErrorContext(ctx, "failed to generate random number", "error", err)
		return errors.New(somethingWentWrongMsg)
	}
	if rnd.Int64() == 0 {
		filter = dal.FindRandomWordFilter{StreakLimitDirection: dal.LimitDirectionLessThan, StreakLimit: 0} // every 100th word to be random
	}
	return b.sendWordCheck(ctx, chatID, filter, &noOpReplier{})
}

func (b *Bot) sendWordCheck(ctx context.Context, chatID int64, filter dal.FindRandomWordFilter, replier replier) error {
	wt, err := b.repo.FindRandomWordTranslation(ctx, chatID, filter)
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

	_, err = b.bot.Send(tb.ChatID(chatID), normalizeMessage(fmt.Sprintf("**%s**", wt.Word)),
		tb.ModeMarkdownV2, tb.Silent, seeTranslationMarkup(callbackID),
	)
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

	if parts[0] == callbackAuthConfirm {
		if err := b.repo.ConfirmAuthConfirmation(ctx, int(c.Chat().ID), parts[1]); err != nil {
			b.log.ErrorContext(ctx, "failed to confirm callback data", "error", err)
			return c.RespondText(somethingWentWrongMsg)
		}

		return c.Delete()
	} else if parts[0] == callbackAuthDecline {
		if err := b.repo.DeleteAuthConfirmation(ctx, int(c.Chat().ID), parts[1]); err != nil {
			b.log.ErrorContext(ctx, "failed to decline callback data", "error", err)
			return c.RespondText(somethingWentWrongMsg)
		}
		return c.Delete()
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
		err = c.Send(normalizeMessage(msg), guessedResponseMarkup(cData.ID), tb.ModeMarkdownV2, tb.Silent)
	case callbackWordGuessed:
		err = b.repo.IncreaseGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordMissed:
		err = b.repo.ResetGuessedStreak(ctx, c.Chat().ID, cData.Word)
	case callbackWordToReview:
		err = b.repo.MarkToReview(ctx, c.Chat().ID, cData.Word)
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

func (b *Bot) HandleDocument(m tb.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	file, err := b.bot.File(&m.Message().Document.File)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to get file", "error", err)
		return m.Reply(somethingWentWrongMsg)
	}

	lines := make(chan data.Line)
	addFailed := false
	parseCtx, parseCancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for line := range lines {
			if gErr := b.repo.AddWordTranslation(ctx, m.Chat().ID, line.Word, line.Translation, line.Description); gErr != nil {
				addFailed = true
				parseCancel()
				b.log.ErrorContext(ctx, "failed to add translation", "error", gErr)
				break
			}
		}
	}()

	err = data.Parse(parseCtx, file, lines)
	wg.Wait()
	if err != nil {
		var pErr *data.ParsingError
		if !errors.As(err, &pErr) {
			b.log.ErrorContext(ctx, "failed to parse document", "error", err)
			return m.Reply(somethingWentWrongMsg)
		}

		invalidLines := pErr.InvalidLines[:int(math.Min(float64(len(pErr.InvalidLines)), 10))] //nolint:mnd // 10 is a magic number
		return m.Reply(fmt.Sprintf("invalid lines=%v", invalidLines))
	}

	if addFailed {
		return m.Reply(somethingWentWrongMsg)
	}

	return m.Reply("translations added")
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
	"#",
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
