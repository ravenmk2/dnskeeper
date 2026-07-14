package service

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

const maxRecordID = 9999

type RecordService struct {
	store store.Store
}

func (s *RecordService) List(ctx context.Context, zone, domain string) ([]store.Record, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	if _, err := s.store.GetDomain(ctx, zone, domain); err != nil {
		return nil, err
	}
	return s.store.ListRecords(ctx, zone, domain)
}

func (s *RecordService) Get(ctx context.Context, zone, domain, id string) (*store.Record, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	if !validRecordIDFormat(id) {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	if _, err := s.store.GetDomain(ctx, zone, domain); err != nil {
		return nil, err
	}
	return s.store.GetRecord(ctx, zone, domain, id)
}

func (s *RecordService) Create(ctx context.Context, zone, domain string, r *store.Record) (*store.Record, error) {
	if !validateZone(zone) || !validateDomain(domain) {
		return nil, apperr.Validation
	}
	if !validateRecordType(r.Type) {
		return nil, apperr.RecordTypeInvalid
	}
	if !validateRecordValue(r.Type, r.Value) {
		return nil, apperr.Validation
	}
	if r.TTL < 1 || r.TTL > 86400 {
		return nil, apperr.Validation
	}
	if err := validateTypeFields(r); err != nil {
		return nil, err
	}
	if r.Type == "SRV" && r.Weight == nil {
		w := 0
		r.Weight = &w
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	prefix := s.store.SkydnsPrefix()
	domainKey := store.DomainKey(zone, domain)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, domainKey)
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
		if d.LastRecordID >= maxRecordID {
			return nil, apperr.RecordIdExhausted
		}
		existing, err := s.store.ListRecords(ctx, zone, domain)
		if err != nil {
			return nil, err
		}
		for i := range existing {
			if recordsDuplicate(r, &existing[i]) {
				return nil, apperr.RecordExists
			}
		}
		newID := d.LastRecordID + 1
		r.ID = formatRecordID(newID)
		d.LastRecordID = newID
		d.RecordCount++
		d.UpdatedAt = now()
		recordData, err := store.MarshalRecord(r)
		if err != nil {
			return nil, err
		}
		domainData, err := store.MarshalDomain(&d)
		if err != nil {
			return nil, err
		}
		skydnsMsg := store.BuildSkydnsMsg(r)
		skydnsData, err := store.MarshalSkydnsMsg(&skydnsMsg)
		if err != nil {
			return nil, err
		}
		ops := []store.Op{
			store.PutOp{Key: store.RecordKey(zone, domain, r.ID), Value: recordData},
			store.PutOp{Key: domainKey, Value: domainData},
			store.PutOp{Key: store.SkydnsRecordKey(prefix, zone, domain, r.ID), Value: skydnsData},
		}
		ok, err := s.store.TxnCAS(ctx, domainKey, kv.ModRevision, ops)
		if err != nil {
			return nil, err
		}
		if ok {
			return r, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *RecordService) Update(ctx context.Context, zone, domain, id string, value *string, ttl *int, priority, port, weight *int) (*store.Record, error) {
	if !validateZone(zone) || !validateDomain(domain) || !validRecordIDFormat(id) {
		return nil, apperr.Validation
	}
	if value == nil && ttl == nil && priority == nil && port == nil && weight == nil {
		return nil, apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return nil, err
	}
	prefix := s.store.SkydnsPrefix()
	domainKey := store.DomainKey(zone, domain)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, domainKey)
		if err != nil {
			return nil, err
		}
		if kv == nil {
			return nil, apperr.DomainNotFound
		}
		existing, err := s.store.GetRecord(ctx, zone, domain, id)
		if err != nil {
			return nil, err
		}
		if priority != nil || port != nil || weight != nil {
			if existing.Type != "SRV" {
				return nil, apperr.Validation
			}
		}
		merged := *existing
		if value != nil {
			if !validateRecordValue(existing.Type, *value) {
				return nil, apperr.Validation
			}
			merged.Value = *value
		}
		if ttl != nil {
			if *ttl < 1 || *ttl > 86400 {
				return nil, apperr.Validation
			}
			merged.TTL = *ttl
		}
		if priority != nil {
			merged.Priority = priority
		}
		if port != nil {
			merged.Port = port
		}
		if weight != nil {
			merged.Weight = weight
		}
		if err := validateTypeFields(&merged); err != nil {
			return nil, err
		}
		records, err := s.store.ListRecords(ctx, zone, domain)
		if err != nil {
			return nil, err
		}
		for i := range records {
			if records[i].ID == id {
				continue
			}
			if recordsDuplicate(&merged, &records[i]) {
				return nil, apperr.RecordExists
			}
		}
		skydnsMsg := store.BuildSkydnsMsg(&merged)
		recordData, err := store.MarshalRecord(&merged)
		if err != nil {
			return nil, err
		}
		skydnsData, err := store.MarshalSkydnsMsg(&skydnsMsg)
		if err != nil {
			return nil, err
		}
		ops := []store.Op{
			store.PutOp{Key: store.RecordKey(zone, domain, id), Value: recordData},
			store.PutOp{Key: store.SkydnsRecordKey(prefix, zone, domain, id), Value: skydnsData},
		}
		ok, err := s.store.TxnCAS(ctx, domainKey, kv.ModRevision, ops)
		if err != nil {
			return nil, err
		}
		if ok {
			return &merged, nil
		}
	}
	return nil, apperr.InternalError
}

func (s *RecordService) Delete(ctx context.Context, zone, domain, id string) error {
	if !validateZone(zone) || !validateDomain(domain) || !validRecordIDFormat(id) {
		return apperr.Validation
	}
	if _, err := s.store.GetZone(ctx, zone); err != nil {
		return err
	}
	prefix := s.store.SkydnsPrefix()
	domainKey := store.DomainKey(zone, domain)
	for retry := 0; retry < 3; retry++ {
		kv, err := s.store.Get(ctx, domainKey)
		if err != nil {
			return err
		}
		if kv == nil {
			return apperr.DomainNotFound
		}
		var d store.Domain
		if err := json.Unmarshal(kv.Value, &d); err != nil {
			return err
		}
		if _, err := s.store.GetRecord(ctx, zone, domain, id); err != nil {
			return err
		}
		if d.RecordCount > 0 {
			d.RecordCount--
		}
		d.UpdatedAt = now()
		domainData, err := store.MarshalDomain(&d)
		if err != nil {
			return err
		}
		ops := []store.Op{
			store.DeleteOp{Key: store.RecordKey(zone, domain, id)},
			store.PutOp{Key: domainKey, Value: domainData},
			store.DeleteOp{Key: store.SkydnsRecordKey(prefix, zone, domain, id)},
		}
		ok, err := s.store.TxnCAS(ctx, domainKey, kv.ModRevision, ops)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return apperr.InternalError
}

func validateTypeFields(r *store.Record) error {
	switch r.Type {
	case "A", "AAAA", "TXT":
		if r.Priority != nil || r.Port != nil || r.Weight != nil {
			return apperr.Validation
		}
	case "SRV":
		if r.Priority == nil || r.Port == nil {
			return apperr.Validation
		}
		if *r.Priority < 0 || *r.Priority > 65535 {
			return apperr.Validation
		}
		if *r.Port < 0 || *r.Port > 65535 {
			return apperr.Validation
		}
		if r.Weight != nil && (*r.Weight < 0 || *r.Weight > 65535) {
			return apperr.Validation
		}
	}
	return nil
}

func validRecordIDFormat(id string) bool {
	if len(id) != 4 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
