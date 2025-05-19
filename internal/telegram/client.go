package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strconv"
)

type (
	SendMessageRequest struct {
		ChatID      int64                `json:"chat_id"`
		Text        string               `json:"text"`
		ReplyMarkup InlineKeyboardMarkup `json:"reply_markup,omitempty"`
	}

	InlineKeyboardMarkup struct {
		InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
	}

	InlineKeyboardButton struct {
		Text         string `json:"text"`
		CallbackData string `json:"callback_data,omitempty"`
	}

	Response struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	Client struct {
		token  string
		client *http.Client
		log    *slog.Logger
	}
)

func NewClient(token string, log *slog.Logger) *Client {
	return &Client{
		token:  token,
		client: http.DefaultClient,
		log:    log,
	}
}

func (c *Client) AskAuthConfirmation(ctx context.Context, chatID int64, token string) error {
	reqBody := &SendMessageRequest{
		ChatID: chatID,
		Text:   "Someone is trying to login to web portal. Do you authorize it?",
		ReplyMarkup: InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{
						Text:         "✅ Yes",
						CallbackData: fmt.Sprintf("callback#auth#confirm:%s", token),
					},
					{
						Text:         "❌ No",
						CallbackData: fmt.Sprintf("callback#auth#decline:%s", token),
					},
				},
			},
		},
	}

	marshal, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token), bytes.NewReader(marshal))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 { //nolint:mnd // ignore mnd
		tags := make([]any, 0, 4) //nolint:mnd // ignore mnd
		tags = append(tags, "status", strconv.Itoa(resp.StatusCode))
		if response, err := httputil.DumpResponse(resp, true); err != nil {
			c.log.DebugContext(ctx, "failed to dump response", "error", err)
		} else {
			tags = append(tags, "response", string(response))
		}
		c.log.ErrorContext(ctx, "unexpected response", tags...)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
