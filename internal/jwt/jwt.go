package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

const (
	ClaimTypeAccess  = "access"
	ClaimTypeRefresh = "refresh"
)

func (m *Manager) Issue(userID, username, userType, tokenType string) (string, error) {
	now := time.Now()
	var ttl time.Duration
	if tokenType == ClaimTypeRefresh {
		ttl = m.refreshTTL
	} else {
		ttl = m.accessTTL
	}
	claims := Claims{
		UserID:   userID,
		Username: username,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   userID,
		},
	}
	claims.RegisteredClaims.Audience = []string{tokenType}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) IssuePair(userID, username, userType string) (string, string, error) {
	access, err := m.Issue(userID, username, userType, ClaimTypeAccess)
	if err != nil {
		return "", "", err
	}
	refresh, err := m.Issue(userID, username, userType, ClaimTypeRefresh)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (m *Manager) ParseRefresh(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if len(claims.Audience) == 0 || claims.Audience[0] != ClaimTypeRefresh {
		return nil, errors.New("not a refresh token")
	}
	return claims, nil
}

func (m *Manager) ParseAccess(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if len(claims.Audience) == 0 || claims.Audience[0] != ClaimTypeAccess {
		return nil, errors.New("not an access token")
	}
	return claims, nil
}
