package telegram

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

const (
	commandStart  = "/start"
	commandStats  = "/stats"
	commandRandom = "/random"

	callbackAuthConfirm    = "callback#auth#confirm"
	callbackAuthDecline    = "callback#auth#decline"
	callbackSeeTranslation = "callback#see_translation"
	callbackWordGuessed    = "callback#word#guessed"
	callbackWordMissed     = "callback#word#missed"
	callbackWordToReview   = "callback#word#to_review"

	somethingWentWrongMsg = "something went wrong"

	processTimeout = 10 * time.Second

	callbackDataExpirationTime = 24 * 7 * time.Hour
)

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
	b.bot.Handle(commandStats, b.HandleStats, b.middlewares...)
	b.bot.Handle(commandRandom, b.HandleRandom, b.middlewares...)
	b.bot.Handle(tb.OnCallback, b.HandleCallback, b.middlewares...)

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

	totalStats, err := b.repo.GetTotalStats(ctx, m.Chat().ID)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to get stats", "error", err)
		return m.Reply("failed to get stats")
	}

	stats, err := b.repo.GetStats(ctx, m.Chat().ID, time.Now())
	if err != nil && !errors.Is(err, dal.ErrNotFound) {
		b.log.ErrorContext(ctx, "failed to get stats", "error", err)
		return m.Reply("failed to get stats")
	}

	msg := fmt.Sprintf("Overall Progress:\n15+: %d\n10-14: %d\n1-9: %d\nTotal: %d",
		totalStats.GreaterThanOrEqual15, totalStats.Between10And14, totalStats.Between1And9, totalStats.Total)

	if stats != nil {
		msg += fmt.Sprintf("\n\nToday's Progress:\nGuessed: %d\nMissed: %d",
			stats.WordsGuessed, stats.WordsMissed)
	}

	return m.Reply(msg)
}

func (b *Bot) HandleRandom(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	return b.sendWordCheck(ctx, m.Chat().ID, dal.FindRandomWordFilter{StreakLimitDirection: dal.LimitDirectionGreaterThanOrEqual, StreakLimit: 0}, m)
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
			return replier.Reply("no words to check") //nolint:wrapcheck // lets ignore it here
		}

		b.log.ErrorContext(ctx, "failed to get random translation", "error", err)
		return replier.Reply(somethingWentWrongMsg) //nolint:wrapcheck // lets ignore it here
	}

	data := dal.CallbackData{
		ChatID:    chatID,
		Word:      wt.Word,
		ExpiresAt: time.Now().Add(callbackDataExpirationTime),
	}
	callbackID, err := b.repo.InsertCallback(ctx, data)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to insert callback data", "error", err)
		return replier.Reply(somethingWentWrongMsg) //nolint:wrapcheck // lets ignore it here
	}

	_, err = b.bot.Send(tb.ChatID(chatID), normalizeMessage(fmt.Sprintf("**%s**", wt.Word)),
		tb.ModeMarkdownV2, tb.Silent, seeTranslationMarkup(callbackID),
	)
	return err //nolint:wrapcheck // lets ignore it here
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
