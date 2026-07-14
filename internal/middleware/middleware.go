package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	KeyRequestID contextKey = "request_id"
	KeyUserID    contextKey = "user_id"
	KeyUsername  contextKey = "username"
	KeyUserType  contextKey = "user_type"
)

func RequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		rid := c.Request().Header.Get("X-Request-Id")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), KeyRequestID, rid)))
		c.Response().Header().Set("X-Request-Id", rid)
		return next(c)
	}
}

func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyRequestID).(string); ok {
		return v
	}
	return ""
}

func Logger(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			rid := GetRequestID(req.Context())
			fields := logrus.Fields{
				"request_id": rid,
				"method":     req.Method,
				"uri":        req.RequestURI,
				"remote":     req.RemoteAddr,
			}
			entry := logger.WithFields(fields)
			err := next(c)
			resp := c.Response()
			status := resp.Status
			if err != nil {
				if ae, ok := apperr.As(err); ok {
					status = ae.HTTPCode
				} else {
					status = http.StatusInternalServerError
				}
			}
			entry.WithField("status", status).Info("request")
			return err
		}
	}
}

func Recover(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.WithFields(logrus.Fields{
						"request_id": GetRequestID(c.Request().Context()),
						"panic":      r,
					}).Errorf("recovered panic: %s", debug.Stack())
					err = apperr.InternalError
				}
			}()
			return next(c)
		}
	}
}

func Auth(jwtManager *jwt.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" || len(auth) < 8 || !strings.EqualFold(auth[:7], "Bearer ") {
				return envelope.Error(c, http.StatusUnauthorized, apperr.Unauthorized)
			}
			tokenStr := auth[7:]
			claims, err := jwtManager.ParseAccess(tokenStr)
			if err != nil {
				return envelope.Error(c, http.StatusUnauthorized, apperr.Unauthorized)
			}
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, KeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, KeyUsername, claims.Username)
			ctx = context.WithValue(ctx, KeyUserType, claims.UserType)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userType, _ := c.Request().Context().Value(KeyUserType).(string)
		if userType != "admin" {
			return envelope.Error(c, http.StatusForbidden, apperr.Forbidden)
		}
		return next(c)
	}
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyUserID).(string); ok {
		return v
	}
	return ""
}
