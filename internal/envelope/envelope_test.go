package envelope_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestData(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	require.NoError(t, envelope.Data(c, map[string]string{"k": "v"}))
	require.Equal(t, http.StatusOK, rec.Code)
	var env envelope.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, env.Success)
	assert.Nil(t, env.Error)
}

func TestPagedPageCount(t *testing.T) {
	cases := []struct {
		page, size, total int
		wantCount          int
	}{
		{1, 10, 0, 0},
		{1, 0, 100, 0},
		{1, 10, 10, 1},
		{1, 10, 11, 2},
		{1, 100, 1, 1},
		{2, 5, 20, 4},
		{1, 5, 21, 5},
		{1, 5, 19, 4},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			require.NoError(t, envelope.Paged(c, nil, tc.page, tc.size, tc.total))
			var env envelope.Envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			require.True(t, env.Success)
			m, ok := env.Data.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, tc.wantCount, int(m["page_count"].(float64)))
			assert.Equal(t, tc.page, int(m["page"].(float64)))
			assert.Equal(t, tc.size, int(m["size"].(float64)))
			assert.Equal(t, tc.total, int(m["total"].(float64)))
		})
	}
}

func TestError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	details := []apperr.Detail{{Code: "X", Message: "m", Target: "f"}}
	ae := apperr.ValidationError(details)
	require.NoError(t, envelope.Error(c, http.StatusBadRequest, ae))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.False(t, env.Success)
	require.NotNil(t, env.Error)
	assert.Equal(t, "VALIDATION_ERROR", env.Error.Code)
	require.Len(t, env.Error.Details, 1)
	assert.Equal(t, "X", env.Error.Details[0].Code)
}

func TestErrorNoDetailsOmitsField(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	require.NoError(t, envelope.Error(c, http.StatusForbidden, apperr.Forbidden))
	require.Equal(t, http.StatusForbidden, rec.Code)
	var env envelope.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "FORBIDDEN", env.Error.Code)
	assert.Nil(t, env.Error.Details)
}
