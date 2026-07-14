package store

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
)

func (s *etcdStore) GetUser(ctx context.Context, id string) (*User, error) {
	kv, err := s.Get(ctx, UserKey(id))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.UserNotFound
	}
	var u User
	if err := json.Unmarshal(kv.Value, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *etcdStore) ListUsers(ctx context.Context) ([]User, error) {
	kvs, err := s.GetPrefix(ctx, UsersPrefix())
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(kvs))
	for _, kv := range kvs {
		var u User
		if err := json.Unmarshal(kv.Value, &u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func MarshalUser(u *User) ([]byte, error) {
	return json.Marshal(u)
}
