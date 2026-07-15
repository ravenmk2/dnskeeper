package service_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

type fakeStore struct {
	mu            sync.Mutex
	data          map[string][]byte
	modRevs       map[string]int64
	modCount      int64
	skydnsPrefix  string

	// Error injection hooks. Set before concurrent goroutines start.
	// nil/zero = no injection (original behavior).
	getErr          error
	getPrefixErr    error
	putErr          error
	deleteErr       error
	deletePrefixErr error
	txnErr          error
	txnCASErr       error
	casFailCount    int // first N TxnCAS calls return (false, nil) to force retry
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		data:         make(map[string][]byte),
		modRevs:      make(map[string]int64),
		skydnsPrefix: "/skydns",
	}
}

func (s *fakeStore) bumpRev(key string) {
	s.modCount++
	s.modRevs[key] = s.modCount
}

func (s *fakeStore) Get(_ context.Context, key string) (*store.KV, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return &store.KV{Key: key, Value: v, ModRevision: s.modRevs[key]}, nil
}

func (s *fakeStore) GetPrefix(_ context.Context, prefix string) ([]store.KV, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getPrefixErr != nil {
		return nil, s.getPrefixErr
	}
	var result []store.KV
	for k, v := range s.data {
		if strings.HasPrefix(k, prefix) {
			result = append(result, store.KV{Key: k, Value: v, ModRevision: s.modRevs[k]})
		}
	}
	return result, nil
}

func (s *fakeStore) Put(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.putErr != nil {
		return s.putErr
	}
	cp := make([]byte, len(value))
	copy(cp, value)
	s.data[key] = cp
	s.bumpRev(key)
	return nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.data, key)
	delete(s.modRevs, key)
	return nil
}

func (s *fakeStore) DeletePrefix(_ context.Context, prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deletePrefixErr != nil {
		return s.deletePrefixErr
	}
	for k := range s.data {
		if strings.HasPrefix(k, prefix) {
			delete(s.data, k)
			delete(s.modRevs, k)
		}
	}
	return nil
}

func (s *fakeStore) Txn(_ context.Context, ops []store.Op) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.txnErr != nil {
		return s.txnErr
	}
	for _, op := range ops {
		switch o := op.(type) {
		case store.PutOp:
			s.applyPut(o)
		case store.DeleteOp:
			s.applyDelete(o)
		case store.DeletePrefixOp:
			s.applyDeletePrefix(o)
		}
	}
	return nil
}

func (s *fakeStore) SkydnsPrefix() string { return s.skydnsPrefix }

func (s *fakeStore) TxnCAS(_ context.Context, key string, modRevision int64, ops []store.Op) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.txnCASErr != nil {
		return false, s.txnCASErr
	}
	if s.casFailCount > 0 {
		s.casFailCount--
		return false, nil
	}
	if s.modRevs[key] != modRevision {
		return false, nil
	}
	for _, op := range ops {
		switch o := op.(type) {
		case store.PutOp:
			s.applyPut(o)
		case store.DeleteOp:
			s.applyDelete(o)
		case store.DeletePrefixOp:
			s.applyDeletePrefix(o)
		}
	}
	return true, nil
}

func (s *fakeStore) applyPut(o store.PutOp) {
	cp := make([]byte, len(o.Value))
	copy(cp, o.Value)
	s.data[o.Key] = cp
	s.bumpRev(o.Key)
}

func (s *fakeStore) applyDelete(o store.DeleteOp) {
	delete(s.data, o.Key)
	delete(s.modRevs, o.Key)
}

func (s *fakeStore) applyDeletePrefix(o store.DeletePrefixOp) {
	for k := range s.data {
		if strings.HasPrefix(k, o.Prefix) {
			delete(s.data, k)
			delete(s.modRevs, k)
		}
	}
}

func (s *fakeStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	kv, err := s.Get(ctx, store.UserKey(id))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.UserNotFound
	}
	var u store.User
	if err := json.Unmarshal(kv.Value, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *fakeStore) ListUsers(ctx context.Context) ([]store.User, error) {
	kvs, err := s.GetPrefix(ctx, store.UsersPrefix())
	if err != nil {
		return nil, err
	}
	users := make([]store.User, 0, len(kvs))
	for _, kv := range kvs {
		var u store.User
		if err := json.Unmarshal(kv.Value, &u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *fakeStore) GetZone(ctx context.Context, zone string) (*store.Zone, error) {
	kv, err := s.Get(ctx, store.ZoneKey(zone))
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
	return &z, nil
}

func (s *fakeStore) ListZones(ctx context.Context) ([]store.Zone, error) {
	kvs, err := s.GetPrefix(ctx, store.PrefixDNS+"/")
	if err != nil {
		return nil, err
	}
	zones := make([]store.Zone, 0, len(kvs))
	for _, kv := range kvs {
		suffix := strings.TrimPrefix(kv.Key, store.PrefixDNS+"/")
		if strings.Contains(suffix, "/") {
			continue
		}
		var z store.Zone
		if err := json.Unmarshal(kv.Value, &z); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}
	return zones, nil
}

func (s *fakeStore) GetDomain(ctx context.Context, zone, domain string) (*store.Domain, error) {
	kv, err := s.Get(ctx, store.DomainKey(zone, domain))
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
	return &d, nil
}

func (s *fakeStore) ListDomains(ctx context.Context, zone string) ([]store.Domain, error) {
	kvs, err := s.GetPrefix(ctx, store.ZonePrefix(zone))
	if err != nil {
		return nil, err
	}
	domains := make([]store.Domain, 0, len(kvs))
	for _, kv := range kvs {
		suffix := strings.TrimPrefix(kv.Key, store.ZonePrefix(zone))
		if strings.Contains(suffix, "/") {
			continue
		}
		var d store.Domain
		if err := json.Unmarshal(kv.Value, &d); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (s *fakeStore) GetRecord(ctx context.Context, zone, domain, id string) (*store.Record, error) {
	kv, err := s.Get(ctx, store.RecordKey(zone, domain, id))
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, apperr.RecordNotFound
	}
	var r store.Record
	if err := json.Unmarshal(kv.Value, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *fakeStore) ListRecords(ctx context.Context, zone, domain string) ([]store.Record, error) {
	kvs, err := s.GetPrefix(ctx, store.DomainPrefix(zone, domain))
	if err != nil {
		return nil, err
	}
	records := make([]store.Record, 0, len(kvs))
	for _, kv := range kvs {
		var r store.Record
		if err := json.Unmarshal(kv.Value, &r); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
