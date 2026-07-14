package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
)

func (h *Handlers) ListUsers(c echo.Context) error {
	users, err := h.svc.User.List(c.Request().Context())
	if err != nil {
		return err
	}
	resp := make([]userResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}
	return envelope.Data(c, resp)
}

type createUserReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	UserType string `json:"user_type" validate:"required"`
}

func (h *Handlers) CreateUser(c echo.Context) error {
	var req createUserReq
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.svc.User.Create(c.Request().Context(), req.Username, req.Password, req.UserType)
	if err != nil {
		return err
	}
	return envelope.Data(c, toUserResponse(user))
}

type updateUserReq struct {
	ID       string `json:"id" validate:"required"`
	Password string `json:"password"`
	UserType string `json:"user_type"`
}

func (h *Handlers) UpdateUser(c echo.Context) error {
	var req updateUserReq
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.svc.User.Update(c.Request().Context(), req.ID, req.Password, req.UserType)
	if err != nil {
		return err
	}
	return envelope.Data(c, toUserResponse(user))
}

type deleteUserReq struct {
	ID string `json:"id" validate:"required"`
}

func (h *Handlers) DeleteUser(c echo.Context) error {
	var req deleteUserReq
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.svc.User.Delete(c.Request().Context(), req.ID); err != nil {
		return err
	}
	return envelope.Data(c, nil)
}
