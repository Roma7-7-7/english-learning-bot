package api

import (
	"net/http"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/labstack/echo/v4"
)

const (
	authCookieName   = "auth"
	accessCookieName = "access"
)

type CookiesProcessor struct {
	path            string
	domain          string
	authExpiresIn   time.Duration
	accessExpiresIn time.Duration
}

func NewCookiesProcessor(conf config.Cookie) *CookiesProcessor {
	return &CookiesProcessor{
		path:            conf.Path,
		domain:          conf.Domain,
		authExpiresIn:   conf.AuthExpiresIn,
		accessExpiresIn: conf.AccessExpiresIn,
	}
}

func (p *CookiesProcessor) NewAuthTokenCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Path:     p.path,
		Domain:   p.domain,
		Value:    token,
		Expires:  time.Now().Add(p.authExpiresIn),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (p *CookiesProcessor) GetAuthToken(c echo.Context) (string, bool) {
	cookie, err := c.Cookie(authCookieName)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

func (p *CookiesProcessor) NewAccessTokenCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     accessCookieName,
		Path:     p.path,
		Domain:   p.domain,
		Value:    token,
		Expires:  time.Now().Add(p.accessExpiresIn),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (p *CookiesProcessor) GetAccessToken(c echo.Context) (string, bool) {
	cookie, err := c.Cookie(accessCookieName)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

func (p *CookiesProcessor) ExpireAuthTokenCookie() *http.Cookie {
	return &http.Cookie{
		Name:    authCookieName,
		Path:    p.path,
		Domain:  p.domain,
		Expires: time.Now(),
	}
}

func (p *CookiesProcessor) ExpireAccessTokenCookie() *http.Cookie {
	return &http.Cookie{
		Name:    accessCookieName,
		Path:    p.path,
		Domain:  p.domain,
		Expires: time.Now(),
	}
}
