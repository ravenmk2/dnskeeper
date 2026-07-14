package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

type DomainService struct {
	store store.Store
}

func (s *DomainService) List(ctx context.Context, zone string) ([]store.Domain, error) {
	if !validateZone(zone) {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	return s.store.ListDomains(ctx, zone)
}

func (s *DomainService) Get(ctx context.Context, zone, domain string) (*store.Domain, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	return s.store.GetDomain(ctx, zone, domain)
}

func (s *DomainService) Create(ctx context.Context, zone, domain string) (*store.Domain, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	zoneKey := store.ZoneKey(zone)
	for retry := 0; retry < 3; retry++ {
		zKv, err := s.store.Get(ctx, zoneKey)
		if err != nil {
			return nil, err
		}
		if zKv == nil {
			return nil, apperr.ZoneNotFound
		}
		var z store.Zone
		if err := json.Unmarshal(zKv.Value, &z); err != nil {
			return nil, err
		}
		existing, err := s.store.GetDomain(ctx, zone, domain)
		if err != nil && !errors.Is(err, apperr.DomainNotFound) {
			return nil, err
		}
		if existing != nil {
			return nil, apperr.DomainExists
		}
		if err := s.checkZoneConflict(ctx, zone, domain); err != nil {
			return nil, err
		}
		ts := now()
		var name string
		if domain == "@" {
			name = zone
		} else {
			name = domain + "." + zone
		}
		d := &store.Domain{
			Zone:         zone,
			Domain:       domain,
			Name:         name,
			RecordCount:  0,
			LastRecordID: 0,
			CreatedAt:    ts,
			UpdatedAt:    ts,
		}
		z.DomainCount++
		domainData, err := store.MarshalDomain(d)
		if err != nil {
			return nil, err
		}
		zoneData, err := store.MarshalZone(&z)
		if err != nil {
			return nil, err
		}
		ops := []store.Op{
			store.PutOp{Key: store.DomainKey(zone, domain), Value: domainData},
			store.PutOp{Key: zoneKey, Value: zoneData},
		}
		ok, err := s.store.TxnCAS(ctx, zoneKey, zKv.ModRevision, ops)
		if err != nil {
			return nil, err
		}
		if ok {
			return d, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *DomainService) Update(ctx context.Context, zone, domain string) (*store.Domain, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	key := store.DomainKey(zone, domain)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if kv == nil {
			return nil, apperr.DomainNotFound
		}
		var d store.Domain
		if err := json.Unmarshal(kv.Value, &d); err != nil {
			return nil, err
		}
		d.UpdatedAt = now()
		data, err := store.MarshalDomain(&d)
		if err != nil {
			return nil, err
		}
		ok, err := s.store.TxnCAS(ctx, key, kv.ModRevision, []store.Op{
			store.PutOp{Key: key, Value: data},
		})
		if err != nil {
			return nil, err
		}
		if ok {
			return &d, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *DomainService) Delete(ctx context.Context, zone, domain string) error {
	if !validateZone(zone) || !validateDomain(domain) {
		return apperr.Validation
	}
	zoneKey := store.ZoneKey(zone)
	prefix := s.store.SkydnsPrefix()
	for retry := 0; retry < 3; retry++ {
		zKv, err := s.store.Get(ctx, zoneKey)
		if err != nil {
			return err
		}
		if zKv == nil {
			return apperr.ZoneNotFound
		}
		var z store.Zone
		if err := json.Unmarshal(zKv.Value, &z); err != nil {
			return err
		}
		d, err := s.store.GetDomain(ctx, zone, domain)
		if err != nil {
			return err
		}
		_ = d
		if z.DomainCount > 0 {
			z.DomainCount--
		}
		zoneData, err := store.MarshalZone(&z)
		if err != nil {
			return err
		}
		ops := []store.Op{
			store.DeletePrefixOp{Prefix: store.DomainPrefix(zone, domain)},
			store.DeleteOp{Key: store.DomainKey(zone, domain)},
			store.PutOp{Key: zoneKey, Value: zoneData},
		}
		if domain == "@" {
			records, err := s.store.ListRecords(ctx, zone, domain)
			if err != nil {
				return err
			}
			for i := range records {
				ops = append(ops, store.DeleteOp{Key: store.SkydnsRecordKey(prefix, zone, domain, records[i].ID)})
			}
		} else {
			ops = append(ops, store.DeletePrefixOp{Prefix: store.SkydnsDomainPrefix(prefix, zone, domain)})
		}
		ok, err := s.store.TxnCAS(ctx, zoneKey, zKv.ModRevision, ops)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return apperr.InternalError
}

func (s *DomainService) checkZoneConflict(ctx context.Context, zone, domain string) error {
	if domain == "@" {
		return nil
	}
	zones, err := s.store.ListZones(ctx)
	if err != nil {
		return err
	}
	zoneSet := make(map[string]bool, len(zones))
	for _, z := range zones {
		zoneSet[z.Zone] = true
	}
	labels := strings.Split(domain, ".")
	fullName := domain + "." + zone
	if zoneSet[fullName] {
		return apperr.DomainZoneConflict
	}
	for i := 1; i < len(labels); i++ {
		ancestor := strings.Join(labels[i:], ".") + "." + zone
		if zoneSet[ancestor] {
			return apperr.DomainZoneConflict
		}
	}
	return nil
}
