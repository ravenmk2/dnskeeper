package store_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/server/v3/embed"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	testClient *clientv3.Client
	testStore  store.Store
	embedSrv   *embed.Etcd
)

func TestMain(m *testing.M) {
	code := 1
	defer func() {
		os.Exit(code)
	}()

	port := freePort()
	peerPort := freePort()
	dataDir := filepath.Join(os.TempDir(), fmt.Sprintf("dnskeeper-test-%d", time.Now().UnixNano()))
	defer os.RemoveAll(dataDir)

	clientURL := url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", port)}
	peerURL := url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", peerPort)}
	clusterStr := fmt.Sprintf("default=%s", peerURL.String())

	cfg := embed.NewConfig()
	cfg.Dir = dataDir
	cfg.ListenClientUrls = []url.URL{clientURL}
	cfg.AdvertiseClientUrls = []url.URL{clientURL}
	cfg.ListenPeerUrls = []url.URL{peerURL}
	cfg.AdvertisePeerUrls = []url.URL{peerURL}
	cfg.InitialCluster = clusterStr
	cfg.LogLevel = "error"

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		fmt.Printf("failed to start embedded etcd: %v\n", err)
		os.Exit(1)
	}
	embedSrv = e
	defer e.Close()

	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(30 * time.Second):
		fmt.Println("embedded etcd startup timeout")
		os.Exit(1)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{fmt.Sprintf("http://127.0.0.1:%d", port)},
	})
	if err != nil {
		fmt.Printf("failed to create etcd client: %v\n", err)
		os.Exit(1)
	}
	testClient = cli
	testStore = store.New(cli, "/skydns")
	defer cli.Close()

	code = m.Run()
}

func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func cleanStore(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := testClient.Delete(ctx, "/dnskeeper/", clientv3.WithPrefix())
	require.NoError(t, err)
	_, err = testClient.Delete(ctx, "/skydns/", clientv3.WithPrefix())
	require.NoError(t, err)
}

func TestStoreUserCRUD(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()

	u := &store.User{
		ID: "admin", Username: "admin", Password: "hashed",
		UserType: "admin", Builtin: true,
		CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z",
	}
	data, err := store.MarshalUser(u)
	require.NoError(t, err)
	require.NoError(t, testStore.Put(ctx, store.UserKey("admin"), data))

	got, err := testStore.GetUser(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Username)
	assert.True(t, got.Builtin)

	users, err := testStore.ListUsers(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 1)

	require.NoError(t, testStore.Delete(ctx, store.UserKey("admin")))
	_, err = testStore.GetUser(ctx, "admin")
	assert.Error(t, err)
}

func TestStoreZoneAndDomain(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()

	z := &store.Zone{Zone: "example.com", DomainCount: 0, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	zdata, _ := store.MarshalZone(z)
	require.NoError(t, testStore.Put(ctx, store.ZoneKey("example.com"), zdata))

	zones, err := testStore.ListZones(ctx)
	require.NoError(t, err)
	assert.Len(t, zones, 1)
	assert.Equal(t, "example.com", zones[0].Zone)

	d := &store.Domain{Zone: "example.com", Domain: "www", Name: "www.example.com", RecordCount: 0, LastRecordID: 0, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	ddata, _ := store.MarshalDomain(d)
	require.NoError(t, testStore.Put(ctx, store.DomainKey("example.com", "www"), ddata))

	domains, err := testStore.ListDomains(ctx, "example.com")
	require.NoError(t, err)
	assert.Len(t, domains, 1)
	assert.Equal(t, "www", domains[0].Domain)

	z2, err := testStore.GetZone(ctx, "example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", z2.Zone)

	d2, err := testStore.GetDomain(ctx, "example.com", "www")
	require.NoError(t, err)
	assert.Equal(t, "www.example.com", d2.Name)
}

func TestStoreRecordAndTxn(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()

	z := &store.Zone{Zone: "example.com", DomainCount: 1, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	zdata, _ := store.MarshalZone(z)
	require.NoError(t, testStore.Put(ctx, store.ZoneKey("example.com"), zdata))

	d := &store.Domain{Zone: "example.com", Domain: "www", Name: "www.example.com", RecordCount: 0, LastRecordID: 0, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	ddata, _ := store.MarshalDomain(d)
	require.NoError(t, testStore.Put(ctx, store.DomainKey("example.com", "www"), ddata))

	r := &store.Record{ID: "0001", Type: "A", Value: "1.2.3.4", TTL: 300}
	rdata, _ := store.MarshalRecord(r)
	prefix := testStore.SkydnsPrefix()
	msg := store.BuildSkydnsMsg(r)
	msgData, _ := store.MarshalSkydnsMsg(&msg)

	ops := []store.Op{
		store.PutOp{Key: store.RecordKey("example.com", "www", "0001"), Value: rdata},
		store.PutOp{Key: store.DomainKey("example.com", "www"), Value: ddata},
		store.PutOp{Key: store.SkydnsRecordKey(prefix, "example.com", "www", "0001"), Value: msgData},
	}
	require.NoError(t, testStore.Txn(ctx, ops))

	got, err := testStore.GetRecord(ctx, "example.com", "www", "0001")
	require.NoError(t, err)
	assert.Equal(t, "A", got.Type)
	assert.Equal(t, "1.2.3.4", got.Value)
	assert.Equal(t, 300, got.TTL)

	records, err := testStore.ListRecords(ctx, "example.com", "www")
	require.NoError(t, err)
	assert.Len(t, records, 1)

	skydnsKey := store.SkydnsRecordKey(prefix, "example.com", "www", "0001")
	skydnsVal, err := testStore.Get(ctx, skydnsKey)
	require.NoError(t, err)
	require.NotNil(t, skydnsVal)
	assert.Contains(t, string(skydnsVal.Value), "1.2.3.4")
	assert.Contains(t, string(skydnsVal.Value), `"ttl":300`)
}

func TestStoreCascadeDelete(t *testing.T) {
	cleanStore(t)
	ctx := context.Background()

	z := &store.Zone{Zone: "example.com", DomainCount: 1, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	zdata, _ := store.MarshalZone(z)
	testStore.Put(ctx, store.ZoneKey("example.com"), zdata)

	d := &store.Domain{Zone: "example.com", Domain: "www", Name: "www.example.com", RecordCount: 1, LastRecordID: 1, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	ddata, _ := store.MarshalDomain(d)
	testStore.Put(ctx, store.DomainKey("example.com", "www"), ddata)

	r := &store.Record{ID: "0001", Type: "A", Value: "1.2.3.4", TTL: 300}
	rdata, _ := store.MarshalRecord(r)
	prefix := testStore.SkydnsPrefix()
	msg := store.BuildSkydnsMsg(r)
	msgData, _ := store.MarshalSkydnsMsg(&msg)
	testStore.Put(ctx, store.RecordKey("example.com", "www", "0001"), rdata)
	testStore.Put(ctx, store.SkydnsRecordKey(prefix, "example.com", "www", "0001"), msgData)

	ops := []store.Op{
		store.DeletePrefixOp{Prefix: store.SkydnsDomainPrefix(prefix, "example.com", "www")},
		store.DeletePrefixOp{Prefix: store.DomainPrefix("example.com", "www")},
		store.DeleteOp{Key: store.DomainKey("example.com", "www")},
	}
	require.NoError(t, testStore.Txn(ctx, ops))

	kv, err := testStore.Get(ctx, store.DomainKey("example.com", "www"))
	require.NoError(t, err)
	assert.Nil(t, kv)

	kv, err = testStore.Get(ctx, store.RecordKey("example.com", "www", "0001"))
	require.NoError(t, err)
	assert.Nil(t, kv)

	kv, err = testStore.Get(ctx, store.SkydnsRecordKey(prefix, "example.com", "www", "0001"))
	require.NoError(t, err)
	assert.Nil(t, kv)
}

func TestSkydnsKeyMapping(t *testing.T) {
	prefix := "/skydns"

	assert.Equal(t, "/skydns/com/example/www/0001",
		store.SkydnsRecordKey(prefix, "example.com", "www", "0001"))
	assert.Equal(t, "/skydns/com/example/0001",
		store.SkydnsRecordKey(prefix, "example.com", "@", "0001"))
	assert.Equal(t, "/skydns/com/example/beta/www/0001",
		store.SkydnsRecordKey(prefix, "example.com", "www.beta", "0001"))
	assert.Equal(t, "/skydns/com/example/sub/api/0001",
		store.SkydnsRecordKey(prefix, "sub.example.com", "api", "0001"))
}
