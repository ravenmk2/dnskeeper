package store_test

import (
	"context"
	"testing"

	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSkydnsMsgAAAA(t *testing.T) {
	r := &store.Record{Type: "AAAA", Value: "::1", TTL: 600}
	msg := store.BuildSkydnsMsg(r)
	assert.Equal(t, "::1", msg.Host)
	assert.Equal(t, 600, msg.TTL)
	assert.Empty(t, msg.Text)
	assert.Nil(t, msg.Priority)
	assert.Nil(t, msg.Port)
	assert.Nil(t, msg.Weight)
}

func TestBuildSkydnsMsgTXT(t *testing.T) {
	r := &store.Record{Type: "TXT", Value: "hello-world", TTL: 300}
	msg := store.BuildSkydnsMsg(r)
	assert.Equal(t, "hello-world", msg.Text)
	assert.Empty(t, msg.Host)
	assert.Nil(t, msg.Priority)
	assert.Nil(t, msg.Port)
	assert.Nil(t, msg.Weight)
}

func TestBuildSkydnsMsgSRV(t *testing.T) {
	pri, port, weight := 10, 5060, 20
	r := &store.Record{Type: "SRV", Value: "srv.example.com", TTL: 300, Priority: &pri, Port: &port, Weight: &weight}
	msg := store.BuildSkydnsMsg(r)
	assert.Equal(t, "srv.example.com", msg.Host)
	require.NotNil(t, msg.Priority)
	assert.Equal(t, pri, *msg.Priority)
	require.NotNil(t, msg.Port)
	assert.Equal(t, port, *msg.Port)
	require.NotNil(t, msg.Weight)
	assert.Equal(t, weight, *msg.Weight)
}

func TestBuildSkydnsMsgUnknownType(t *testing.T) {
	r := &store.Record{Type: "UNKNOWN", Value: "x", TTL: 100}
	msg := store.BuildSkydnsMsg(r)
	assert.Equal(t, 100, msg.TTL)
	assert.Empty(t, msg.Host)
	assert.Empty(t, msg.Text)
	assert.Nil(t, msg.Priority)
}

func TestSkydnsZonePrefixMapping(t *testing.T) {
	assert.Equal(t, "/skydns/com/example/", store.SkydnsZonePrefix("/skydns", "example.com"))
	assert.Equal(t, "/skydns/org/example/sub/", store.SkydnsZonePrefix("/skydns", "sub.example.org"))
	assert.Equal(t, "/skydns/com/example/", store.SkydnsDomainPrefix("/skydns", "example.com", "@"))
	assert.Equal(t, "/skydns/com/example/www/", store.SkydnsDomainPrefix("/skydns", "example.com", "www"))
}

func TestStoreTxnCASSuccess(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	key := store.ZoneKey("example.com")
	require.NoError(t, testStore.Put(ctx, key, []byte(`{"zone":"example.com"}`)))
	kv, err := testStore.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, kv)

	ok, err := testStore.TxnCAS(ctx, key, kv.ModRevision, []store.Op{
		store.PutOp{Key: key, Value: []byte(`{"zone":"example.com","domain_count":1}`)},
	})
	require.NoError(t, err)
	assert.True(t, ok)

	kv2, err := testStore.Get(ctx, key)
	require.NoError(t, err)
	assert.Contains(t, string(kv2.Value), "domain_count")
	assert.Greater(t, kv2.ModRevision, kv.ModRevision)
}

func TestStoreTxnCASMismatch(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	key := store.ZoneKey("example.com")
	require.NoError(t, testStore.Put(ctx, key, []byte(`{"zone":"example.com"}`)))

	ok, err := testStore.TxnCAS(ctx, key, 999999, []store.Op{
		store.PutOp{Key: key, Value: []byte(`{"zone":"overridden"}`)},
	})
	require.NoError(t, err)
	assert.False(t, ok)

	kv, err := testStore.Get(ctx, key)
	require.NoError(t, err)
	assert.Contains(t, string(kv.Value), "example.com")
	assert.NotContains(t, string(kv.Value), "overridden")
}

func TestStoreTxnCASCreateNew(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	key := store.ZoneKey("new.com")
	ok, err := testStore.TxnCAS(ctx, key, 0, []store.Op{
		store.PutOp{Key: key, Value: []byte(`{"zone":"new.com"}`)},
	})
	require.NoError(t, err)
	assert.True(t, ok)

	kv, err := testStore.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, kv)
	assert.Greater(t, kv.ModRevision, int64(0))
}

func TestStoreTxnCASCreateConflict(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	key := store.ZoneKey("exists.com")
	require.NoError(t, testStore.Put(ctx, key, []byte(`{"zone":"exists.com"}`)))

	ok, err := testStore.TxnCAS(ctx, key, 0, []store.Op{
		store.PutOp{Key: key, Value: []byte(`{"zone":"override"}`)},
	})
	require.NoError(t, err)
	assert.False(t, ok)

	kv, err := testStore.Get(ctx, key)
	require.NoError(t, err)
	assert.Contains(t, string(kv.Value), "exists.com")
	assert.NotContains(t, string(kv.Value), "override")
}

func TestStoreDeletePrefixInterface(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	require.NoError(t, testStore.Put(ctx, "/dnskeeper/dns/z1", []byte("a")))
	require.NoError(t, testStore.Put(ctx, "/dnskeeper/dns/z2", []byte("b")))
	require.NoError(t, testStore.Put(ctx, "/dnskeeper/users/u1", []byte("c")))

	require.NoError(t, testStore.DeletePrefix(ctx, "/dnskeeper/dns/"))

	kv, err := testStore.Get(ctx, "/dnskeeper/dns/z1")
	require.NoError(t, err)
	assert.Nil(t, kv)
	kv, err = testStore.Get(ctx, "/dnskeeper/dns/z2")
	require.NoError(t, err)
	assert.Nil(t, kv)
	// dns 前缀外不受影响
	kv, err = testStore.Get(ctx, "/dnskeeper/users/u1")
	require.NoError(t, err)
	require.NotNil(t, kv)
	assert.Equal(t, "c", string(kv.Value))
}

func TestStoreGetPrefixDirect(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()
	require.NoError(t, testStore.Put(ctx, store.UserKey("a"), []byte("1")))
	require.NoError(t, testStore.Put(ctx, store.UserKey("b"), []byte("2")))

	kvs, err := testStore.GetPrefix(ctx, store.UsersPrefix())
	require.NoError(t, err)
	assert.Len(t, kvs, 2)
}
