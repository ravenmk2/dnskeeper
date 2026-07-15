package apperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorString(t *testing.T) {
	ae := apperr.New("CODE", "msg", 500)
	assert.Equal(t, "CODE: msg", ae.Error())
}

func TestAsDirect(t *testing.T) {
	ae := apperr.New("CODE", "msg", 500)
	got, ok := apperr.As(ae)
	require.True(t, ok)
	assert.Equal(t, ae, got)
}

func TestAsWrapped(t *testing.T) {
	wrapped := fmt.Errorf("lookup: %w", apperr.UserNotFound)
	got, ok := apperr.As(wrapped)
	require.True(t, ok)
	assert.Equal(t, apperr.UserNotFound, got)
}

func TestAsWrappedTwice(t *testing.T) {
	inner := fmt.Errorf("ctx: %w", apperr.ZoneExists)
	outer := fmt.Errorf("outer: %w", inner)
	got, ok := apperr.As(outer)
	require.True(t, ok)
	assert.Equal(t, apperr.ZoneExists, got)
}

func TestAsNotAppError(t *testing.T) {
	_, ok := apperr.As(errors.New("plain"))
	assert.False(t, ok)
}

func TestValidationError(t *testing.T) {
	details := []apperr.Detail{{Code: "X", Message: "m", Target: "f"}}
	ae := apperr.ValidationError(details)
	assert.Equal(t, "VALIDATION_ERROR", ae.Code)
	assert.Equal(t, http.StatusOK, ae.HTTPCode)
	assert.Equal(t, details, ae.Details)
}

func TestWithDetails(t *testing.T) {
	details := []apperr.Detail{{Code: "C", Message: "mm", Target: "t"}}
	ae := apperr.WithDetails("CODE", "msg", http.StatusBadRequest, details)
	assert.Equal(t, http.StatusBadRequest, ae.HTTPCode)
	assert.Equal(t, details, ae.Details)
}

func TestSentinelHTTPCodes(t *testing.T) {
	cases := []struct {
		err  *apperr.AppError
		code int
	}{
		{apperr.Validation, http.StatusOK},
		{apperr.InvalidCredentials, http.StatusUnauthorized},
		{apperr.WrongPassword, http.StatusOK},
		{apperr.SamePassword, http.StatusOK},
		{apperr.WeakPassword, http.StatusOK},
		{apperr.Unauthorized, http.StatusUnauthorized},
		{apperr.InvalidToken, http.StatusUnauthorized},
		{apperr.Forbidden, http.StatusForbidden},
		{apperr.CannotDeleteBuiltin, http.StatusOK},
		{apperr.CannotDemoteBuiltin, http.StatusOK},
		{apperr.UserNotFound, http.StatusOK},
		{apperr.UserExists, http.StatusOK},
		{apperr.ZoneNotFound, http.StatusOK},
		{apperr.ZoneExists, http.StatusOK},
		{apperr.DomainNotFound, http.StatusOK},
		{apperr.DomainExists, http.StatusOK},
		{apperr.DomainZoneConflict, http.StatusOK},
		{apperr.RecordNotFound, http.StatusOK},
		{apperr.RecordExists, http.StatusOK},
		{apperr.RecordTypeInvalid, http.StatusOK},
		{apperr.RecordIdExhausted, http.StatusOK},
		{apperr.ServiceUnavailable, http.StatusServiceUnavailable},
		{apperr.InternalError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.err.Code, func(t *testing.T) {
			assert.Equal(t, tc.code, tc.err.HTTPCode)
		})
	}
}
