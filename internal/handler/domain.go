package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
)

type zoneReq struct {
	Zone string `json:"zone" validate:"required"`
}

func (h *Handlers) ListDomains(c echo.Context) error {
	var req zoneReq
	if err := bind(c, &req); err != nil {
		return err
	}
	domains, err := h.svc.Domain.List(c.Request().Context(), req.Zone)
	if err != nil {
		return err
	}
	resp := make([]domainResponse, 0, len(domains))
	for i := range domains {
		resp = append(resp, toDomainResponse(&domains[i]))
	}
	return envelope.Data(c, resp)
}

type domainKeyReq struct {
	Zone   string `json:"zone" validate:"required"`
	Domain string `json:"domain" validate:"required"`
}

func (h *Handlers) GetDomain(c echo.Context) error {
	var req domainKeyReq
	if err := bind(c, &req); err != nil {
		return err
	}
	d, err := h.svc.Domain.Get(c.Request().Context(), req.Zone, req.Domain)
	if err != nil {
		return err
	}
	return envelope.Data(c, toDomainResponse(d))
}

func (h *Handlers) CreateDomain(c echo.Context) error {
	var req domainKeyReq
	if err := bind(c, &req); err != nil {
		return err
	}
	d, err := h.svc.Domain.Create(c.Request().Context(), req.Zone, req.Domain)
	if err != nil {
		return err
	}
	return envelope.Data(c, toDomainResponse(d))
}

func (h *Handlers) UpdateDomain(c echo.Context) error {
	var req domainKeyReq
	if err := bind(c, &req); err != nil {
		return err
	}
	d, err := h.svc.Domain.Update(c.Request().Context(), req.Zone, req.Domain)
	if err != nil {
		return err
	}
	return envelope.Data(c, toDomainResponse(d))
}

func (h *Handlers) DeleteDomain(c echo.Context) error {
	var req domainKeyReq
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.svc.Domain.Delete(c.Request().Context(), req.Zone, req.Domain); err != nil {
		return err
	}
	return envelope.Data(c, nil)
}
