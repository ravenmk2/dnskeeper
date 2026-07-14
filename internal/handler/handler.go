package handler

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/service"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	svc   *service.Services
	store store.Store
}

func New(svc *service.Services, s store.Store) *Handlers {
	return &Handlers{svc: svc, store: s}
}

type userResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	UserType  string `json:"user_type"`
	Builtin   bool   `json:"builtin"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toUserResponse(u *store.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		UserType:  u.UserType,
		Builtin:   u.Builtin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type domainResponse struct {
	Zone        string `json:"zone"`
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	RecordCount int    `json:"record_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toDomainResponse(d *store.Domain) domainResponse {
	return domainResponse{
		Zone:        d.Zone,
		Domain:      d.Domain,
		Name:        d.Name,
		RecordCount: d.RecordCount,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func bind(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return apperr.Validation
	}
	if err := c.Validate(req); err != nil {
		var ves validator.ValidationErrors
		if errors.As(err, &ves) {
			details := make([]apperr.Detail, 0, len(ves))
			for _, ve := range ves {
				details = append(details, apperr.Detail{
					Code:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("field '%s' failed '%s'", ve.Field(), ve.Tag()),
					Target:  ve.Field(),
				})
			}
			return apperr.ValidationError(details)
		}
		return apperr.Validation
	}
	return nil
}
