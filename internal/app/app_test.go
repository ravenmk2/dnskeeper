package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSilentLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func newCtx() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

type errorEnv struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestErrorHandlerAppError(t *testing.T) {
	c, rec := newCtx()
	errorHandler(apperr.Forbidden, c)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var env errorEnv
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.False(t, env.Success)
	assert.Equal(t, "FORBIDDEN", env.Error.Code)
}

func TestErrorHandlerHTTPErrorNotFound(t *testing.T) {
	c, rec := newCtx()
	errorHandler(&echo.HTTPError{Code: http.StatusNotFound, Message: "nope"}, c)
	require.Equal(t, http.StatusNotFound, rec.Code)
	var env errorEnv
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
}

func TestErrorHandlerHTTPErrorMethodNotAllowed(t *testing.T) {
	c, rec := newCtx()
	errorHandler(&echo.HTTPError{Code: http.StatusMethodNotAllowed}, c)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	var env errorEnv
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "METHOD_NOT_ALLOWED", env.Error.Code)
}

func TestErrorHandlerHTTPErrorDefault(t *testing.T) {
	c, rec := newCtx()
	errorHandler(&echo.HTTPError{Code: http.StatusBadGateway}, c)
	require.Equal(t, http.StatusBadGateway, rec.Code)
	var env errorEnv
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "INTERNAL_ERROR", env.Error.Code)
}

func TestErrorHandlerHTTPErrorZeroCode(t *testing.T) {
	c, rec := newCtx()
	errorHandler(&echo.HTTPError{Code: 0}, c)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestErrorHandlerGenericError(t *testing.T) {
	c, rec := newCtx()
	errorHandler(errors.New("boom"), c)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var env errorEnv
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "INTERNAL_ERROR", env.Error.Code)
}

type stubStore struct {
	store.Store
	user    *store.User
	getErr  error
	putErr  error
	putKey  string
	putVal  []byte
}

func (s *stubStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.user == nil {
		return nil, apperr.UserNotFound
	}
	return s.user, nil
}

func (s *stubStore) Put(ctx context.Context, key string, value []byte) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.putKey = key
	s.putVal = value
	return nil
}

func TestSeedAdminAlreadyExists(t *testing.T) {
	st := &stubStore{user: &store.User{ID: "admin"}}
	a := &App{store: st, logger: newSilentLogger()}
	require.NoError(t, a.SeedAdmin(context.Background()))
	assert.Empty(t, st.putKey)
}

func TestSeedAdminCreates(t *testing.T) {
	st := &stubStore{}
	a := &App{store: st, logger: newSilentLogger()}
	require.NoError(t, a.SeedAdmin(context.Background()))
	assert.Equal(t, store.UserKey("admin"), st.putKey)
	var u store.User
	require.NoError(t, json.Unmarshal(st.putVal, &u))
	assert.Equal(t, "admin", u.ID)
	assert.Equal(t, "admin", u.Username)
	assert.Equal(t, "admin", u.UserType)
	assert.True(t, u.Builtin)
}

func TestSeedAdminGetError(t *testing.T) {
	st := &stubStore{getErr: errors.New("etcd down")}
	a := &App{store: st, logger: newSilentLogger()}
	err := a.SeedAdmin(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd down")
}

func TestSeedAdminPutError(t *testing.T) {
	st := &stubStore{putErr: errors.New("put failed")}
	a := &App{store: st, logger: newSilentLogger()}
	err := a.SeedAdmin(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "put failed")
}
