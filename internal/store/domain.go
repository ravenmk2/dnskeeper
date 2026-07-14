package store

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
)

func (s *etcdStore) GetDomain(ctx context.Context, zone, domain string) (*Domain, error) {
	kv, err := s.Get(ctx, DomainKey(zone, domain))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.DomainNotFound
	}
	var d Domain
	if err := json.Unmarshal(kv.Value, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *etcdStore) ListDomains(ctx context.Context, zone string) ([]Domain, error) {
	kvs, err := s.GetPrefix(ctx, ZonePrefix(zone))
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, 0, len(kvs))
	for _, kv := range kvs {
		suffix := stripPrefix(string(kv.Key), ZonePrefix(zone))
		if !isLeaf(suffix) {
			continue
		}
		var d Domain
		if err := json.Unmarshal(kv.Value, &d); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func MarshalDomain(d *Domain) ([]byte, error) {
	return json.Marshal(d)
}
