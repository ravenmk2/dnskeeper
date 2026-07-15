package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/middleware"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEcho() *echo.Echo { return echo.New() }

func newSilentLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestRequestIDPassthrough(t *testing.T) {
	e := newEcho()
	e.Use(middleware.RequestID)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, middleware.GetRequestID(c.Request().Context()))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "rid-123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "rid-123", rec.Body.String())
	assert.Equal(t, "rid-123", rec.Header().Get("X-Request-Id"))
}

func TestRequestIDGenerated(t *testing.T) {
	e := newEcho()
	e.Use(middleware.RequestID)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, middleware.GetRequestID(c.Request().Context()))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	rid := rec.Body.String()
	assert.NotEmpty(t, rid)
	assert.Len(t, rid, 36)
	assert.Equal(t, rid, rec.Header().Get("X-Request-Id"))
}

func TestGetRequestIDMissing(t *testing.T) {
	assert.Equal(t, "", middleware.GetRequestID(context.Background()))
}

func TestGetUserIDMissing(t *testing.T) {
	assert.Equal(t, "", middleware.GetUserID(context.Background()))
}

func TestAuthInvalidBearer(t *testing.T) {
	mgr := jwt.NewManager("s", time.Minute, time.Hour)
	cases := []string{
		"",        // 缺失
		"Bearer",  // 无空格
		"Basic xyz",
		"bearer ",  // 正好 7 字符,被 len<8 拦截
		"Bearer ",  // 正好 7 字符,被 len<8 拦截
		"Beare xyz",
	}
	for _, authz := range cases {
		t.Run(authz, func(t *testing.T) {
			e := newEcho()
			e.Use(middleware.Auth(mgr))
			e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if authz != "" {
				req.Header.Set("Authorization", authz)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "Authorization=%q", authz)
		})
	}
}

func TestAuthInvalidToken(t *testing.T) {
	mgr := jwt.NewManager("s", time.Minute, time.Hour)
	e := newEcho()
	e.Use(middleware.Auth(mgr))
	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthValidTokenInjectsContext(t *testing.T) {
	mgr := jwt.NewManager("s", time.Minute, time.Hour)
	access, _, err := mgr.IssuePair("u-1", "alice", "admin")
	require.NoError(t, err)

	var gotID, gotName, gotType string
	e := newEcho()
	e.Use(middleware.Auth(mgr))
	e.GET("/", func(c echo.Context) error {
		ctx := c.Request().Context()
		gotID, _ = ctx.Value(middleware.KeyUserID).(string)
		gotName, _ = ctx.Value(middleware.KeyUsername).(string)
		gotType, _ = ctx.Value(middleware.KeyUserType).(string)
		return c.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "u-1", gotID)
	assert.Equal(t, "alice", gotName)
	assert.Equal(t, "admin", gotType)
}

func TestRequireAdmin(t *testing.T) {
	cases := []struct {
		name     string
		userType string
		set      bool
		want     int
	}{
		{"admin", "admin", true, http.StatusOK},
		{"user", "user", true, http.StatusForbidden},
		{"empty", "", true, http.StatusForbidden},
		{"missing", "", false, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEcho()
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					if tc.set {
						ctx := context.WithValue(c.Request().Context(), middleware.KeyUserType, tc.userType)
						c.SetRequest(c.Request().WithContext(ctx))
					}
					return next(c)
				}
			}, middleware.RequireAdmin)
			e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code)
		})
	}
}

func TestRecover(t *testing.T) {
	e := newEcho()
	e.Use(middleware.Recover(newSilentLogger()))
	e.GET("/", func(c echo.Context) error { panic("boom") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestLoggerOK(t *testing.T) {
	e := newEcho()
	e.Use(middleware.RequestID, middleware.Logger(newSilentLogger()))
	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// Logger 仅用 apperr.As 推断 status 用于日志,不影响响应状态码(由 echo error handler 决定)。
// 因此通过捕获日志输出来验证其内部 status 推断的两个分支。
func newCapturingLogger() (*logrus.Logger, *bytes.Buffer) {
	l := logrus.New()
	buf := &bytes.Buffer{}
	l.SetOutput(buf)
	l.SetFormatter(&logrus.JSONFormatter{})
	return l, buf
}

func TestLoggerAppErrorStatus(t *testing.T) {
	logger, buf := newCapturingLogger()
	e := newEcho()
	e.Use(middleware.RequestID, middleware.Logger(logger))
	e.GET("/", func(c echo.Context) error { return apperr.InvalidCredentials })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Contains(t, buf.String(), `"status":401`)
}

func TestLoggerGenericErrorStatus(t *testing.T) {
	logger, buf := newCapturingLogger()
	e := newEcho()
	e.Use(middleware.RequestID, middleware.Logger(logger))
	e.GET("/", func(c echo.Context) error { return errors.New("boom") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Contains(t, buf.String(), `"status":500`)
}
