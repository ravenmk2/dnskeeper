package service

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

type ZoneService struct {
	store store.Store
}

func (s *ZoneService) List(ctx context.Context) ([]store.Zone, error) {
	return s.store.ListZones(ctx)
}

func (s *ZoneService) Get(ctx context.Context, zone string) (*store.Zone, error) {
	if !validateZone(zone) {
		return nil, apperr.Validation
	}
	return s.store.GetZone(ctx, zone)
}

func (s *ZoneService) Create(ctx context.Context, zone string) (*store.Zone, error) {
	if !validateZone(zone) {
		return nil, apperr.Validation
	}
	key := store.ZoneKey(zone)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if kv != nil {
			return nil, apperr.ZoneExists
		}
		ts := now()
		z := &store.Zone{
			Zone:        zone,
			DomainCount: 0,
			CreatedAt:   ts,
			UpdatedAt:   ts,
		}
		data, err := store.MarshalZone(z)
		if err != nil {
			return nil, err
		}
		ok, err := s.store.TxnCAS(ctx, key, 0, []store.Op{
			store.PutOp{Key: key, Value: data},
		})
		if err != nil {
			return nil, err
		}
		if ok {
			return z, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *ZoneService) Update(ctx context.Context, zone string) (*store.Zone, error) {
	if !validateZone(zone) {
		return nil, apperr.Validation
	}
	key := store.ZoneKey(zone)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if kv == nil {
			return nil, apperr.ZoneNotFound
		}
		var z store.Zone
		if err := json.Unmarshal(kv.Value, &z); err != nil {
			return nil, err
		}
		z.UpdatedAt = now()
		data, err := store.MarshalZone(&z)
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
			return &z, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *ZoneService) Delete(ctx context.Context, zone string) error {
	if !validateZone(zone) {
		return apperr.Validation
	}
	z, err := s.store.GetZone(ctx, zone)
	if err != nil {
		return err
	}
	_ = z
	ops := []store.Op{
		store.DeletePrefixOp{Prefix: store.SkydnsZonePrefix(s.store.SkydnsPrefix(), zone)},
		store.DeletePrefixOp{Prefix: store.ZonePrefix(zone)},
		store.DeleteOp{Key: store.ZoneKey(zone)},
	}
	return s.store.Txn(ctx, ops)
}
