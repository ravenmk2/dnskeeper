package store

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
)

func (s *etcdStore) GetRecord(ctx context.Context, zone, domain, id string) (*Record, error) {
	kv, err := s.Get(ctx, RecordKey(zone, domain, id))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.RecordNotFound
	}
	var r Record
	if err := json.Unmarshal(kv.Value, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *etcdStore) ListRecords(ctx context.Context, zone, domain string) ([]Record, error) {
	kvs, err := s.GetPrefix(ctx, DomainPrefix(zone, domain))
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(kvs))
	for _, kv := range kvs {
		var r Record
		if err := json.Unmarshal(kv.Value, &r); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func MarshalRecord(r *Record) ([]byte, error) {
	return json.Marshal(r)
}
