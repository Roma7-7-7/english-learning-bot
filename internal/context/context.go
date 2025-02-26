package context

import "context"

type chatIDKey struct{}

func WithChatID(ctx context.Context, chatID int64) context.Context {
	return context.WithValue(ctx, chatIDKey{}, chatID)
}

func ChatIDFromContext(ctx context.Context) (int64, bool) {
	chatID, ok := ctx.Value(chatIDKey{}).(int64)
	return chatID, ok
}
