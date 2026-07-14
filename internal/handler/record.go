package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

type domainKeyReq2 struct {
	Zone   string `json:"zone" validate:"required"`
	Domain string `json:"domain" validate:"required"`
}

func (h *Handlers) ListRecords(c echo.Context) error {
	var req domainKeyReq2
	if err := bind(c, &req); err != nil {
		return err
	}
	records, err := h.svc.Record.List(c.Request().Context(), req.Zone, req.Domain)
	if err != nil {
		return err
	}
	return envelope.Data(c, records)
}

type recordKeyReq struct {
	Zone   string `json:"zone" validate:"required"`
	Domain string `json:"domain" validate:"required"`
	ID     string `json:"id" validate:"required"`
}

func (h *Handlers) GetRecord(c echo.Context) error {
	var req recordKeyReq
	if err := bind(c, &req); err != nil {
		return err
	}
	r, err := h.svc.Record.Get(c.Request().Context(), req.Zone, req.Domain, req.ID)
	if err != nil {
		return err
	}
	return envelope.Data(c, r)
}

type createRecordReq struct {
	Zone     string `json:"zone" validate:"required"`
	Domain   string `json:"domain" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Value    string `json:"value" validate:"required"`
	TTL      int    `json:"ttl" validate:"min=1,max=86400"`
	Priority *int   `json:"priority"`
	Port     *int   `json:"port"`
	Weight   *int   `json:"weight"`
}

func (h *Handlers) CreateRecord(c echo.Context) error {
	var req createRecordReq
	if err := bind(c, &req); err != nil {
		return err
	}
	r := &store.Record{
		Type:     req.Type,
		Value:    req.Value,
		TTL:      req.TTL,
		Priority: req.Priority,
		Port:     req.Port,
		Weight:   req.Weight,
	}
	result, err := h.svc.Record.Create(c.Request().Context(), req.Zone, req.Domain, r)
	if err != nil {
		return err
	}
	return envelope.Data(c, result)
}

type updateRecordReq struct {
	Zone     string  `json:"zone" validate:"required"`
	Domain   string  `json:"domain" validate:"required"`
	ID       string  `json:"id" validate:"required"`
	Value    *string `json:"value"`
	TTL      *int    `json:"ttl"`
	Priority *int    `json:"priority"`
	Port     *int    `json:"port"`
	Weight   *int    `json:"weight"`
}

func (h *Handlers) UpdateRecord(c echo.Context) error {
	var req updateRecordReq
	if err := bind(c, &req); err != nil {
		return err
	}
	result, err := h.svc.Record.Update(c.Request().Context(), req.Zone, req.Domain, req.ID, req.Value, req.TTL, req.Priority, req.Port, req.Weight)
	if err != nil {
		return err
	}
	return envelope.Data(c, result)
}

type deleteRecordReq struct {
	Zone   string `json:"zone" validate:"required"`
	Domain string `json:"domain" validate:"required"`
	ID     string `json:"id" validate:"required"`
}

func (h *Handlers) DeleteRecord(c echo.Context) error {
	var req deleteRecordReq
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.svc.Record.Delete(c.Request().Context(), req.Zone, req.Domain, req.ID); err != nil {
		return err
	}
	return envelope.Data(c, nil)
}
