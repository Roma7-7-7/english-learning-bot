package api

import (
	"fmt"
	"strconv"
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
		Username: strconv.FormatInt(chatID, 10),
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

func (p *JWTProcessor) ParseAuthToken(token string) (int64, string, error) {
	var parsed *jwt.Token
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return p.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return 0, "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return 0, "", fmt.Errorf("invalid token claims")
	}

	// Validate issuer and audience
	if iss, _ := claims.GetIssuer(); iss != p.issuer {
		return 0, "", fmt.Errorf("invalid issuer")
	}
	if aud, _ := claims.GetAudience(); !containsAll(aud, p.audience) {
		return 0, "", fmt.Errorf("invalid audience")
	}

	subject, err := parsed.Claims.GetSubject()
	if err != nil {
		return 0, "", fmt.Errorf("get subject: %w", err)
	}
	var chatID int64
	var key string
	_, err = fmt.Sscanf(subject, "%d:%s", &chatID, &key)
	if err != nil {
		return 0, "", fmt.Errorf("parse subject: %w", err)
	}
	return chatID, key, nil
}

func (p *JWTProcessor) ToAccessToken(chatID int64) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Username: strconv.FormatInt(chatID, 10),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuer,
			Subject:   strconv.FormatInt(chatID, 10),
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

func (p *JWTProcessor) ParseAccessToken(token string) (int64, error) {
	var parsed *jwt.Token
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return p.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return 0, fmt.Errorf("invalid token claims")
	}

	// Validate issuer and audience
	if iss, _ := claims.GetIssuer(); iss != p.issuer {
		return 0, fmt.Errorf("invalid issuer")
	}
	if aud, _ := claims.GetAudience(); !containsAll(aud, p.audience) {
		return 0, fmt.Errorf("invalid audience")
	}

	subject, err := parsed.Claims.GetSubject()
	if err != nil {
		return 0, fmt.Errorf("get subject: %w", err)
	}
	var chatID int64
	_, err = fmt.Sscanf(subject, "%d", &chatID)
	if err != nil {
		return 0, fmt.Errorf("parse subject: %w", err)
	}
	return chatID, nil
}

// containsAll returns true if all elements in required are present in actual
func containsAll(actual, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(actual) < len(required) {
		return false
	}
	for _, r := range required {
		found := false
		for _, a := range actual {
			if a == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
