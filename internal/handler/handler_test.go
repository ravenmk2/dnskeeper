package handler

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vadapter struct{ v *validator.Validate }

func (a *vadapter) Validate(i interface{}) error { return a.v.Struct(i) }

func newJSONContext(body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestBindValid(t *testing.T) {
	c, _ := newJSONContext(`{"zone":"z","domain":"d","type":"A","value":"1.2.3.4","ttl":300}`)
	c.Echo().Validator = &vadapter{validator.New()}
	var req createRecordReq
	require.NoError(t, bind(c, &req))
	assert.Equal(t, "z", req.Zone)
	assert.Equal(t, "d", req.Domain)
	assert.Equal(t, "A", req.Type)
	assert.Equal(t, "1.2.3.4", req.Value)
	assert.Equal(t, 300, req.TTL)
}

func TestBindMalformedJSON(t *testing.T) {
	c, _ := newJSONContext(`{bad json`)
	var req createRecordReq
	err := bind(c, &req)
	require.Error(t, err)
	assert.Equal(t, apperr.Validation, err)
}

func TestBindValidationErrors(t *testing.T) {
	c, _ := newJSONContext(`{"ttl":0}`)
	c.Echo().Validator = &vadapter{validator.New()}
	var req createRecordReq
	err := bind(c, &req)
	require.Error(t, err)
	ae, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", ae.Code)
	assert.NotEmpty(t, ae.Details)
	for _, d := range ae.Details {
		assert.Equal(t, "VALIDATION_ERROR", d.Code)
		assert.NotEmpty(t, d.Message)
		assert.NotEmpty(t, d.Target)
	}
}

func TestBindRecordTTLBoundary(t *testing.T) {
	cases := []struct {
		ttl  int
		want bool
	}{
		{0, false},
		{1, true},
		{86400, true},
		{86401, false},
		{-1, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("ttl_%d", tc.ttl), func(t *testing.T) {
			body := fmt.Sprintf(`{"zone":"z","domain":"d","type":"A","value":"1.2.3.4","ttl":%d}`, tc.ttl)
			c, _ := newJSONContext(body)
			c.Echo().Validator = &vadapter{validator.New()}
			var req createRecordReq
			err := bind(c, &req)
			if tc.want {
				assert.NoError(t, err, "ttl=%d", tc.ttl)
			} else {
				assert.Error(t, err, "ttl=%d", tc.ttl)
			}
		})
	}
}

type errValidator struct{}

func (errValidator) Validate(i interface{}) error { return errors.New("boom") }

func TestBindNonValidatorError(t *testing.T) {
	c, _ := newJSONContext(`{"zone":"z"}`)
	c.Echo().Validator = &errValidator{}
	var req createRecordReq
	err := bind(c, &req)
	require.Error(t, err)
	assert.Equal(t, apperr.Validation, err)
}

func TestToUserResponse(t *testing.T) {
	u := &store.User{
		ID:        "u-1",
		Username:  "alice",
		UserType:  "admin",
		Builtin:   true,
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}
	r := toUserResponse(u)
	assert.Equal(t, u.ID, r.ID)
	assert.Equal(t, u.Username, r.Username)
	assert.Equal(t, u.UserType, r.UserType)
	assert.Equal(t, u.Builtin, r.Builtin)
	assert.Equal(t, u.CreatedAt, r.CreatedAt)
	assert.Equal(t, u.UpdatedAt, r.UpdatedAt)
}

func TestToDomainResponse(t *testing.T) {
	d := &store.Domain{
		Zone:         "z",
		Domain:       "d",
		Name:         "d.z",
		RecordCount:  3,
		LastRecordID: 3,
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-02T00:00:00Z",
	}
	r := toDomainResponse(d)
	assert.Equal(t, d.Zone, r.Zone)
	assert.Equal(t, d.Domain, r.Domain)
	assert.Equal(t, d.Name, r.Name)
	assert.Equal(t, d.RecordCount, r.RecordCount)
	assert.Equal(t, d.CreatedAt, r.CreatedAt)
	assert.Equal(t, d.UpdatedAt, r.UpdatedAt)
}
