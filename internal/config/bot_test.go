package config

import (
	"context"
	"strings"
	"testing"
)

// setRequired fills in everything validateBot insists on, so that each test can focus on one knob.
func setRequired(t *testing.T) {
	t.Helper()

	t.Setenv("BOT_TELEGRAM_TOKEN", "token")
	t.Setenv("BOT_TELEGRAM_ALLOWED_CHAT_IDS", "1")
	t.Setenv("BOT_HTTP_JWT_SECRET", "secret")
	t.Setenv("BOT_HTTP_JWT_AUDIENCE", "http://localhost")
	t.Setenv("BOT_HTTP_CORS_ALLOW_ORIGINS", "http://localhost")
	t.Setenv("BOT_HTTP_COOKIE_DOMAIN", "localhost")
}

func TestGetBotLearningDefaults(t *testing.T) {
	setRequired(t)

	conf, err := GetBot(context.Background())
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}

	if conf.Learning.BatchSize != 50 {
		t.Errorf("BatchSize = %d, want 50", conf.Learning.BatchSize)
	}
	if conf.Learning.StreakLimit != 15 {
		t.Errorf("StreakLimit = %d, want 15", conf.Learning.StreakLimit)
	}
}

func TestGetBotLearningFromEnv(t *testing.T) {
	setRequired(t)
	t.Setenv("BOT_LEARNING_BATCH_SIZE", "80")
	t.Setenv("BOT_LEARNING_STREAK_LIMIT", "8")

	conf, err := GetBot(context.Background())
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}

	if conf.Learning.BatchSize != 80 {
		t.Errorf("BatchSize = %d, want 80", conf.Learning.BatchSize)
	}
	if conf.Learning.StreakLimit != 8 {
		t.Errorf("StreakLimit = %d, want 8", conf.Learning.StreakLimit)
	}
}

func TestGetBotRejectsInvalidLearningConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "zero batch size",
			env:     map[string]string{"BOT_LEARNING_BATCH_SIZE": "0"},
			wantErr: "learning batch size",
		},
		{
			name:    "negative streak limit",
			env:     map[string]string{"BOT_LEARNING_STREAK_LIMIT": "-1"},
			wantErr: "learning streak limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequired(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := GetBot(context.Background())
			if err == nil {
				t.Fatal("GetBot succeeded, want validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}
