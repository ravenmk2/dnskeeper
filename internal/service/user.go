package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	store store.Store
}

func (s *UserService) List(ctx context.Context) ([]store.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *UserService) Create(ctx context.Context, username, password, userType string) (*store.User, error) {
	if !validateUsername(username) {
		return nil, apperr.Validation
	}
	if !validatePassword(password) {
		return nil, apperr.WeakPassword
	}
	if userType != "admin" && userType != "normal" {
		return nil, apperr.Validation
	}
	id := strings.ToLower(username)
	existing, err := s.store.GetUser(ctx, id)
	if err != nil && !errors.Is(err, apperr.UserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, apperr.UserExists
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	ts := now()
	user := &store.User{
		ID:        id,
		Username:  username,
		Password:  string(hashed),
		UserType:  userType,
		Builtin:   false,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	data, err := store.MarshalUser(user)
	if err != nil {
		return nil, err
	}
	if err := s.store.Put(ctx, store.UserKey(id), data); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id, password, userType string) (*store.User, error) {
	id = strings.ToLower(id)
	hasPassword := password != ""
	hasUserType := userType != ""
	if !hasPassword && !hasUserType {
		return nil, apperr.Validation
	}
	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if hasUserType {
		if userType != "admin" && userType != "normal" {
			return nil, apperr.Validation
		}
		if user.Builtin && userType == "normal" {
			return nil, apperr.CannotDemoteBuiltin
		}
		user.UserType = userType
	}
	if hasPassword {
		if !validatePassword(password) {
			return nil, apperr.WeakPassword
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
	}
	user.UpdatedAt = now()
	data, err := store.MarshalUser(user)
	if err != nil {
		return nil, err
	}
	if err := s.store.Put(ctx, store.UserKey(id), data); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	id = strings.ToLower(id)
	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if user.Builtin {
		return apperr.CannotDeleteBuiltin
	}
	return s.store.Delete(ctx, store.UserKey(id))
}
