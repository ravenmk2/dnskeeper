package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
)

func (h *Handlers) ListZones(c echo.Context) error {
	zones, err := h.svc.Zone.List(c.Request().Context())
	if err != nil {
		return err
	}
	return envelope.Data(c, zones)
}

type getZoneReq struct {
	Zone string `json:"zone" validate:"required"`
}

func (h *Handlers) GetZone(c echo.Context) error {
	var req getZoneReq
	if err := bind(c, &req); err != nil {
		return err
	}
	zone, err := h.svc.Zone.Get(c.Request().Context(), req.Zone)
	if err != nil {
		return err
	}
	return envelope.Data(c, zone)
}

type createZoneReq struct {
	Zone string `json:"zone" validate:"required"`
}

func (h *Handlers) CreateZone(c echo.Context) error {
	var req createZoneReq
	if err := bind(c, &req); err != nil {
		return err
	}
	zone, err := h.svc.Zone.Create(c.Request().Context(), req.Zone)
	if err != nil {
		return err
	}
	return envelope.Data(c, zone)
}

func (h *Handlers) UpdateZone(c echo.Context) error {
	var req createZoneReq
	if err := bind(c, &req); err != nil {
		return err
	}
	zone, err := h.svc.Zone.Update(c.Request().Context(), req.Zone)
	if err != nil {
		return err
	}
	return envelope.Data(c, zone)
}

func (h *Handlers) DeleteZone(c echo.Context) error {
	var req createZoneReq
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.svc.Zone.Delete(c.Request().Context(), req.Zone); err != nil {
		return err
	}
	return envelope.Data(c, nil)
}
