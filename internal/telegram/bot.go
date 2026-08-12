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

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
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

	// reviewPrefix marks a word that is being re-tested after having been learned, so it is obvious
	// that a wrong answer will cost a streak that was already complete.
	reviewPrefix = "🔁 "
)

type (
	Bot struct {
		bot  *tb.Bot
		repo dal.Repository

		// streakLimit is the streak at which a word counts as learned and becomes eligible for
		// review; reviewRatePercent is the share of scheduled checks spent on those reviews.
		streakLimit       int
		reviewRatePercent int

		middlewares []tb.MiddlewareFunc

		log *slog.Logger
	}

	replier interface {
		Reply(any, ...any) error
	}

	noOpReplier struct{}
)

func NewBot(token string, repo dal.Repository, conf config.Learning, log *slog.Logger, middlewares ...tb.MiddlewareFunc) (*Bot, error) {
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
		bot:               b,
		repo:              repo,
		streakLimit:       conf.StreakLimit,
		reviewRatePercent: conf.ReviewRatePercent,
		middlewares:       middlewares,
		log:               log,
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

	msg := totalStatsMessage(totalStats)

	if stats != nil {
		msg += fmt.Sprintf("\n\nToday's Progress:\nGuessed: %d\nMissed: %d",
			stats.WordsGuessed, stats.WordsMissed)
	}

	return m.Reply(msg)
}

// totalStatsMessage renders the overall progress breakdown, skipping the buckets that a small
// streak limit squeezes out of existence: with a limit of 6 or less the early band has no room left
// and would be labelled with the inverted range "1-0".
func totalStatsMessage(s *dal.TotalStats) string {
	lines := []string{
		"Overall Progress:",
		fmt.Sprintf("%d+: %d", s.StreakLimit, s.Learned),
	}
	if s.StreakLimit-1 >= s.NearlyFrom {
		lines = append(lines, fmt.Sprintf("%d-%d: %d", s.NearlyFrom, s.StreakLimit-1, s.Nearly))
	}
	if s.NearlyFrom > 1 {
		lines = append(lines, fmt.Sprintf("1-%d: %d", s.NearlyFrom-1, s.Early))
	}
	lines = append(lines, fmt.Sprintf("Total: %d", s.Total))
	lines = append(lines, fmt.Sprintf("In learning batch: %d", s.Batched))

	return strings.Join(lines, "\n")
}

func (b *Bot) HandleRandom(m tb.Context) error {
	ctx, cancel := processCtx()
	defer cancel()

	return b.sendWordCheck(ctx, m.Chat().ID, dal.FindRandomWordFilter{StreakLimitDirection: dal.LimitDirectionGreaterThanOrEqual, StreakLimit: 0}, m)
}

// SendWordCheck sends one scheduled word check.
//
// Most checks come from the active learning batch, but ReviewRatePercent of them re-test a word that
// has already been learned. Without that, a word never comes back once its streak crosses the limit,
// so the "learned" count drifts away from what is actually remembered.
func (b *Bot) SendWordCheck(ctx context.Context, chatID int64) error {
	// Any failure to pick a review falls back to the batch, same as a check that was never going to
	// be a review: pickReview has already logged whatever went wrong, and losing the review is
	// better than losing the whole check.
	review, err := b.pickReview(ctx, chatID)
	if err != nil {
		return b.sendWordCheck(ctx, chatID, dal.FindRandomWordFilter{Batched: true}, &noOpReplier{})
	}

	if err = b.sendWord(ctx, chatID, review, reviewPrefix); err != nil {
		return err
	}
	// Stamped on send rather than on answer, so an ignored message still advances the rotation.
	if err = b.repo.MarkWordReviewed(ctx, chatID, review.Word); err != nil {
		b.log.ErrorContext(ctx, "failed to mark word reviewed", "error", err, "word", review.Word)
	}
	return nil
}

// errNoReviewDue means this check must fall back to the active learning batch: either the draw did
// not land on a review, or there is nothing learned to review yet.
var errNoReviewDue = errors.New("no review due")

// pickReview returns a learned word to re-test, or errNoReviewDue when the check belongs to the
// active batch instead.
func (b *Bot) pickReview(ctx context.Context, chatID int64) (*dal.WordTranslation, error) {
	if b.reviewRatePercent <= 0 {
		return nil, errNoReviewDue
	}

	rnd, err := rand.Int(rand.Reader, big.NewInt(100)) //nolint:mnd // percentages are out of 100
	if err != nil {
		b.log.ErrorContext(ctx, "failed to generate random number", "error", err)
		return nil, errors.New(somethingWentWrongMsg)
	}
	if rnd.Int64() >= int64(b.reviewRatePercent) {
		return nil, errNoReviewDue
	}

	wt, err := b.repo.FindRandomWordTranslation(ctx, chatID, dal.FindRandomWordFilter{
		StreakLimitDirection: dal.LimitDirectionGreaterThanOrEqual,
		StreakLimit:          b.streakLimit,
		Order:                dal.OrderLeastRecentlyReviewed,
	})
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			b.log.DebugContext(ctx, "no learned words to review", "chat_id", chatID)
			return nil, errNoReviewDue
		}
		b.log.ErrorContext(ctx, "failed to get word to review", "error", err)
		return nil, errors.New(somethingWentWrongMsg)
	}
	return wt, nil
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

	// A failed send has to reach the caller: on the scheduled path the replier is a no-op, so
	// returning its nil would hide every delivery failure — including the blocked-user case the
	// scheduler reports separately.
	if err = b.sendWord(ctx, chatID, wt, ""); err != nil {
		b.log.ErrorContext(ctx, "failed to send word check", "error", err, "chat_id", chatID)
		if replyErr := replier.Reply(somethingWentWrongMsg); replyErr != nil {
			b.log.ErrorContext(ctx, "failed to reply", "error", replyErr, "chat_id", chatID)
		}
		return err
	}
	return nil
}

func (b *Bot) sendWord(ctx context.Context, chatID int64, wt *dal.WordTranslation, prefix string) error {
	data := dal.CallbackData{
		ChatID:    chatID,
		Word:      wt.Word,
		ExpiresAt: time.Now().Add(callbackDataExpirationTime),
	}
	callbackID, err := b.repo.InsertCallback(ctx, data)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to insert callback data", "error", err)
		return fmt.Errorf("insert callback data: %w", err)
	}

	_, err = b.bot.Send(tb.ChatID(chatID), prefix+normalizeMessage(fmt.Sprintf("**%s**", wt.Word)),
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
