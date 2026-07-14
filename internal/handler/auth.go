package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

type loginReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type tokenPair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handlers) Login(c echo.Context) error {
	var req loginReq
	if err := bind(c, &req); err != nil {
		return err
	}
	access, refresh, err := h.svc.Auth.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return err
	}
	return envelope.Data(c, tokenPair{Token: access, RefreshToken: refresh})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *Handlers) Refresh(c echo.Context) error {
	var req refreshReq
	if err := bind(c, &req); err != nil {
		return err
	}
	access, refresh, err := h.svc.Auth.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	return envelope.Data(c, tokenPair{Token: access, RefreshToken: refresh})
}

func (h *Handlers) Health(c echo.Context) error {
	ctx := c.Request().Context()
	if _, err := h.store.GetPrefix(ctx, store.PrefixUsers+"/"); err != nil {
		return envelope.Error(c, http.StatusServiceUnavailable, apperr.ServiceUnavailable)
	}
	return envelope.Data(c, nil)
}
