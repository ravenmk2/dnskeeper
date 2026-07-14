package store

import (
	"context"
	"encoding/json"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
)

func (s *etcdStore) GetZone(ctx context.Context, zone string) (*Zone, error) {
	kv, err := s.Get(ctx, ZoneKey(zone))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.ZoneNotFound
	}
	var z Zone
	if err := json.Unmarshal(kv.Value, &z); err != nil {
		return nil, err
	}
	return &z, nil
}

func (s *etcdStore) ListZones(ctx context.Context) ([]Zone, error) {
	kvs, err := s.GetPrefix(ctx, PrefixDNS+"/")
	if err != nil {
		return nil, err
	}
	zones := make([]Zone, 0, len(kvs))
	for _, kv := range kvs {
		suffix := stripPrefix(string(kv.Key), PrefixDNS+"/")
		if !isLeaf(suffix) {
			continue
		}
		var z Zone
		if err := json.Unmarshal(kv.Value, &z); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}
	return zones, nil
}

func MarshalZone(z *Zone) ([]byte, error) {
	return json.Marshal(z)
}
