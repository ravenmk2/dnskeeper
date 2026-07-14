package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	store store.Store
	jwt   *jwt.Manager
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, string, error) {
	id := lowerUsername(username)
	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, apperr.UserNotFound) {
			return "", "", apperr.InvalidCredentials
		}
		return "", "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", "", apperr.InvalidCredentials
	}
	return s.jwt.IssuePair(user.ID, user.Username, user.UserType)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*store.User, error) {
	return s.store.GetUser(ctx, userID)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := s.jwt.ParseRefresh(refreshToken)
	if err != nil {
		return "", "", apperr.InvalidToken
	}
	user, err := s.store.GetUser(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, apperr.UserNotFound) {
			return "", "", apperr.InvalidToken
		}
		return "", "", err
	}
	return s.jwt.IssuePair(user.ID, user.Username, user.UserType)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)) != nil {
		return apperr.WrongPassword
	}
	if oldPassword == newPassword {
		return apperr.SamePassword
	}
	if !validatePassword(newPassword) {
		return apperr.WeakPassword
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)
	user.UpdatedAt = now()
	data, err := store.MarshalUser(user)
	if err != nil {
		return err
	}
	return s.store.Put(ctx, store.UserKey(user.ID), data)
}

func lowerUsername(username string) string {
	return strings.ToLower(username)
}
