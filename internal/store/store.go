package store

import (
	"context"
	"encoding/json"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	PrefixDnskeeper = "/dnskeeper"
	PrefixDNS       = "/dnskeeper/dns"
	PrefixUsers     = "/dnskeeper/users"
)

type KV struct {
	Key         string
	Value       []byte
	ModRevision int64
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	UserType  string `json:"user_type"`
	Builtin   bool   `json:"builtin"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Zone struct {
	Zone        string `json:"zone"`
	DomainCount int    `json:"domain_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Domain struct {
	Zone         string `json:"zone"`
	Domain       string `json:"domain"`
	Name         string `json:"name"`
	RecordCount  int    `json:"record_count"`
	LastRecordID int    `json:"last_record_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Record struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
}

func UserKey(id string) string {
	return PrefixUsers + "/" + id
}

func UsersPrefix() string {
	return PrefixUsers + "/"
}

func ZoneKey(zone string) string {
	return PrefixDNS + "/" + zone
}

func ZonePrefix(zone string) string {
	return ZoneKey(zone) + "/"
}

func DomainKey(zone, domain string) string {
	return PrefixDNS + "/" + zone + "/" + domain
}

func DomainPrefix(zone, domain string) string {
	return DomainKey(zone, domain) + "/"
}

func RecordKey(zone, domain, id string) string {
	return PrefixDNS + "/" + zone + "/" + domain + "/" + id
}

type Op interface {
	toClientOp() clientv3.Op
}

type PutOp struct {
	Key   string
	Value []byte
}

func (o PutOp) toClientOp() clientv3.Op {
	return clientv3.OpPut(o.Key, string(o.Value))
}

type DeleteOp struct {
	Key string
}

func (o DeleteOp) toClientOp() clientv3.Op {
	return clientv3.OpDelete(o.Key)
}

type DeletePrefixOp struct {
	Prefix string
}

func (o DeletePrefixOp) toClientOp() clientv3.Op {
	return clientv3.OpDelete(o.Prefix, clientv3.WithPrefix())
}

type Store interface {
	Get(ctx context.Context, key string) (*KV, error)
	GetPrefix(ctx context.Context, prefix string) ([]KV, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
	Txn(ctx context.Context, ops []Op) error
	TxnCAS(ctx context.Context, key string, modRevision int64, ops []Op) (bool, error)
	SkydnsPrefix() string

	GetUser(ctx context.Context, id string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	GetZone(ctx context.Context, zone string) (*Zone, error)
	ListZones(ctx context.Context) ([]Zone, error)
	GetDomain(ctx context.Context, zone, domain string) (*Domain, error)
	ListDomains(ctx context.Context, zone string) ([]Domain, error)
	GetRecord(ctx context.Context, zone, domain, id string) (*Record, error)
	ListRecords(ctx context.Context, zone, domain string) ([]Record, error)
}

type etcdStore struct {
	client       *clientv3.Client
	skydnsPrefix string
}

func New(client *clientv3.Client, skydnsPrefix string) Store {
	return &etcdStore{client: client, skydnsPrefix: skydnsPrefix}
}

func (s *etcdStore) SkydnsPrefix() string {
	return s.skydnsPrefix
}

func (s *etcdStore) Get(ctx context.Context, key string) (*KV, error) {
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return &KV{Key: string(resp.Kvs[0].Key), Value: resp.Kvs[0].Value, ModRevision: resp.Kvs[0].ModRevision}, nil
}

func (s *etcdStore) GetPrefix(ctx context.Context, prefix string) ([]KV, error) {
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	result := make([]KV, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result = append(result, KV{Key: string(kv.Key), Value: kv.Value, ModRevision: kv.ModRevision})
	}
	return result, nil
}

func (s *etcdStore) Put(ctx context.Context, key string, value []byte) error {
	_, err := s.client.Put(ctx, key, string(value))
	return err
}

func (s *etcdStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.Delete(ctx, key)
	return err
}

func (s *etcdStore) DeletePrefix(ctx context.Context, prefix string) error {
	_, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	return err
}

func (s *etcdStore) Txn(ctx context.Context, ops []Op) error {
	clientOps := make([]clientv3.Op, 0, len(ops))
	for _, op := range ops {
		clientOps = append(clientOps, op.toClientOp())
	}
	_, err := s.client.Txn(ctx).Then(clientOps...).Commit()
	return err
}

func (s *etcdStore) TxnCAS(ctx context.Context, key string, modRevision int64, ops []Op) (bool, error) {
	clientOps := make([]clientv3.Op, 0, len(ops))
	for _, op := range ops {
		clientOps = append(clientOps, op.toClientOp())
	}
	resp, err := s.client.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", modRevision)).
		Then(clientOps...).
		Commit()
	if err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

func unmarshal[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func stripPrefix(key, prefix string) string {
	return strings.TrimPrefix(key, prefix)
}

func isLeaf(suffix string) bool {
	return !strings.Contains(suffix, "/")
}
