package web

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
)

type (
	JWTProcessor struct {
		issuer         string
		audience       []string
		authExpireIn   time.Duration
		accessExpireIn time.Duration

		secret []byte
	}

	Claims struct {
		Username string `json:"username"`
		jwt.RegisteredClaims
	}
)

func NewJWTProcessor(conf config.JWT, authExpireIn, accessExpireIn time.Duration) *JWTProcessor {
	return &JWTProcessor{
		issuer:         conf.Issuer,
		audience:       conf.Audience,
		authExpireIn:   authExpireIn,
		accessExpireIn: accessExpireIn,

		secret: []byte(conf.Secret),
	}
}

func (p *JWTProcessor) ToAuthToken(chatID int64, key string) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Username: fmt.Sprintf("%d", chatID),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuer,
			Subject:   fmt.Sprintf("%d:%s", chatID, key),
			Audience:  p.audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(p.accessExpireIn)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	})

	signedString, err := token.SignedString(p.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedString, nil
}

func (p *JWTProcessor) ParseAuthToken(token string) (chatID int64, key string, err error) {
	var parsed *jwt.Token
	parsed, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return p.secret, nil
	})
	if err != nil {
		return 0, "", fmt.Errorf("parse token: %w", err)
	}
	subject, err := parsed.Claims.GetSubject()
	if err != nil {
		return 0, "", fmt.Errorf("get subject: %w", err)
	}
	_, err = fmt.Sscanf(subject, "%d:%s", &chatID, &key)
	if err != nil {
		return 0, "", fmt.Errorf("parse subject: %w", err)
	}
	return
}

func (p *JWTProcessor) ToAccessToken(chatID int64) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Username: fmt.Sprintf("%d", chatID),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuer,
			Subject:   fmt.Sprintf("%d", chatID),
			Audience:  p.audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(p.accessExpireIn)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	})

	signedString, err := token.SignedString(p.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedString, nil
}

func (p *JWTProcessor) ParseAccessToken(token string) (chatID int64, err error) {
	var parsed *jwt.Token
	parsed, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return p.secret, nil
	})
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}
	subject, err := parsed.Claims.GetSubject()
	if err != nil {
		return 0, fmt.Errorf("get subject: %w", err)
	}
	_, err = fmt.Sscanf(subject, "%d", &chatID)
	if err != nil {
		return 0, fmt.Errorf("parse subject: %w", err)
	}
	return
}
