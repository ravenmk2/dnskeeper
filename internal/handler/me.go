package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/ravenmk2/dnskeeper/internal/middleware"
)

func (h *Handlers) Me(c echo.Context) error {
	userID := middleware.GetUserID(c.Request().Context())
	user, err := h.svc.Auth.GetUser(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return envelope.Data(c, toUserResponse(user))
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

func (h *Handlers) ChangePassword(c echo.Context) error {
	var req changePasswordReq
	if err := bind(c, &req); err != nil {
		return err
	}
	userID := middleware.GetUserID(c.Request().Context())
	if err := h.svc.Auth.ChangePassword(c.Request().Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		return err
	}
	return envelope.Data(c, nil)
}
