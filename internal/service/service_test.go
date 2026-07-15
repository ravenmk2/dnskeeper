package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/service"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestServices() (*service.Services, *fakeStore) {
	fs := newFakeStore()
	jwtMgr := jwt.NewManager("test-secret", 30*time.Minute, 168*time.Hour)
	return service.NewServices(fs, jwtMgr), fs
}

func seedUser(t *testing.T, fs *fakeStore, username, password, userType string, builtin bool) *store.User {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	ts := time.Now().UTC().Format(time.RFC3339)
	u := &store.User{
		ID:        lower(username),
		Username:  username,
		Password:  string(hashed),
		UserType:  userType,
		Builtin:   builtin,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	data, _ := store.MarshalUser(u)
	fs.Put(context.Background(), store.UserKey(u.ID), data)
	return u
}

func lower(s string) string {
	if s == "" {
		return ""
	}
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}

func TestLogin(t *testing.T) {
	svc, fs := newTestServices()
	seedUser(t, fs, "admin", "admin123", "admin", true)

	t.Run("success", func(t *testing.T) {
		access, refresh, err := svc.Auth.Login(context.Background(), "admin", "admin123")
		require.NoError(t, err)
		assert.NotEmpty(t, access)
		assert.NotEmpty(t, refresh)
	})

	t.Run("wrong_password", func(t *testing.T) {
		_, _, err := svc.Auth.Login(context.Background(), "admin", "wrong")
		assert.ErrorIs(t, err, apperr.InvalidCredentials)
	})

	t.Run("user_not_found", func(t *testing.T) {
		_, _, err := svc.Auth.Login(context.Background(), "nobody", "pass")
		assert.ErrorIs(t, err, apperr.InvalidCredentials)
	})
}

func TestChangePassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "user1", "oldpass1", "normal", false)
		err := svc.Auth.ChangePassword(context.Background(), "user1", "oldpass1", "newpass2")
		require.NoError(t, err)
	})

	t.Run("wrong_old", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "user1", "oldpass1", "normal", false)
		err := svc.Auth.ChangePassword(context.Background(), "user1", "wrong", "newpass2")
		assert.ErrorIs(t, err, apperr.WrongPassword)
	})

	t.Run("same_password", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "user1", "oldpass1", "normal", false)
		err := svc.Auth.ChangePassword(context.Background(), "user1", "oldpass1", "oldpass1")
		assert.ErrorIs(t, err, apperr.SamePassword)
	})

	t.Run("weak_new", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "user1", "oldpass1", "normal", false)
		err := svc.Auth.ChangePassword(context.Background(), "user1", "oldpass1", "weak")
		assert.ErrorIs(t, err, apperr.WeakPassword)
	})
}

func TestCreateUser(t *testing.T) {
	svc, _ := newTestServices()

	t.Run("success", func(t *testing.T) {
		u, err := svc.User.Create(context.Background(), "newuser", "Pass1234", "normal")
		require.NoError(t, err)
		assert.Equal(t, "newuser", u.ID)
		assert.False(t, u.Builtin)
	})

	t.Run("exists", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "alice", "Pass1234", "normal", false)
		_, err := svc.User.Create(context.Background(), "alice", "Pass1234", "normal")
		assert.ErrorIs(t, err, apperr.UserExists)
	})

	t.Run("case_insensitive_exists", func(t *testing.T) {
		svc, fs := newTestServices()
		seedUser(t, fs, "alice", "Pass1234", "normal", false)
		_, err := svc.User.Create(context.Background(), "Alice", "Pass1234", "normal")
		assert.ErrorIs(t, err, apperr.UserExists)
	})

	t.Run("weak_password", func(t *testing.T) {
		_, err := svc.User.Create(context.Background(), "bob", "weak", "normal")
		assert.ErrorIs(t, err, apperr.WeakPassword)
	})

	t.Run("invalid_username", func(t *testing.T) {
		_, err := svc.User.Create(context.Background(), "ab", "Pass1234", "normal")
		assert.ErrorIs(t, err, apperr.Validation)
	})
}

func TestDeleteBuiltinUser(t *testing.T) {
	svc, fs := newTestServices()
	seedUser(t, fs, "admin", "admin123", "admin", true)
	err := svc.User.Delete(context.Background(), "admin")
	assert.ErrorIs(t, err, apperr.CannotDeleteBuiltin)
}

func TestUpdateBuiltinUserCannotDemote(t *testing.T) {
	svc, fs := newTestServices()
	seedUser(t, fs, "admin", "admin123", "admin", true)
	_, err := svc.User.Update(context.Background(), "admin", "", "normal")
	assert.ErrorIs(t, err, apperr.CannotDemoteBuiltin)
}

func TestZoneCRUD(t *testing.T) {
	svc, _ := newTestServices()

	t.Run("create_and_get", func(t *testing.T) {
		z, err := svc.Zone.Create(context.Background(), "example.com")
		require.NoError(t, err)
		assert.Equal(t, "example.com", z.Zone)
		assert.Equal(t, 0, z.DomainCount)

		got, err := svc.Zone.Get(context.Background(), "example.com")
		require.NoError(t, err)
		assert.Equal(t, "example.com", got.Zone)
	})

	t.Run("create_exists", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		_, err := svc.Zone.Create(context.Background(), "example.com")
		assert.ErrorIs(t, err, apperr.ZoneExists)
	})

	t.Run("delete_cascade", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

		err := svc.Zone.Delete(context.Background(), "example.com")
		require.NoError(t, err)

		kv, _ := fs.Get(context.Background(), store.ZoneKey("example.com"))
		assert.Nil(t, kv)
		kv, _ = fs.Get(context.Background(), store.DomainKey("example.com", "www"))
		assert.Nil(t, kv)
		kv, _ = fs.Get(context.Background(), store.RecordKey("example.com", "www", "0001"))
		assert.Nil(t, kv)
		kv, _ = fs.Get(context.Background(), store.SkydnsRecordKey("/skydns", "example.com", "www", "0001"))
		assert.Nil(t, kv)
	})
}

func TestDomainZoneConflict(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedZone(fs, "beta.example.com")

	_, err := svc.Domain.Create(context.Background(), "example.com", "www.beta")
	assert.ErrorIs(t, err, apperr.DomainZoneConflict)
}

func TestDomainCRUD(t *testing.T) {
	t.Run("create_and_get", func(t *testing.T) {
		svc, _ := newTestServices()
		_, _ = svc.Zone.Create(context.Background(), "example.com")
		d, err := svc.Domain.Create(context.Background(), "example.com", "www")
		require.NoError(t, err)
		assert.Equal(t, "www.example.com", d.Name)
		assert.Equal(t, 0, d.RecordCount)

		got, err := svc.Domain.Get(context.Background(), "example.com", "www")
		require.NoError(t, err)
		assert.Equal(t, "www", got.Domain)
	})

	t.Run("create_at_root", func(t *testing.T) {
		svc, _ := newTestServices()
		_, _ = svc.Zone.Create(context.Background(), "example.com")
		d, err := svc.Domain.Create(context.Background(), "example.com", "@")
		require.NoError(t, err)
		assert.Equal(t, "example.com", d.Name)
	})

	t.Run("delete_cascade_records", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

		err := svc.Domain.Delete(context.Background(), "example.com", "www")
		require.NoError(t, err)

		kv, _ := fs.Get(context.Background(), store.DomainKey("example.com", "www"))
		assert.Nil(t, kv)
		kv, _ = fs.Get(context.Background(), store.RecordKey("example.com", "www", "0001"))
		assert.Nil(t, kv)
		kv, _ = fs.Get(context.Background(), store.SkydnsRecordKey("/skydns", "example.com", "www", "0001"))
		assert.Nil(t, kv)
	})

	t.Run("delete_root_domain_preserves_other_domains", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "@")
		seedDomain(fs, "example.com", "www")
		seedRecord(fs, "example.com", "@", "0001", "A", "1.2.3.4")
		seedRecord(fs, "example.com", "www", "0001", "A", "5.6.7.8")

		err := svc.Domain.Delete(context.Background(), "example.com", "@")
		require.NoError(t, err)

		kv, _ := fs.Get(context.Background(), store.DomainKey("example.com", "@"))
		assert.Nil(t, kv)

		kv, _ = fs.Get(context.Background(), store.SkydnsRecordKey("/skydns", "example.com", "@", "0001"))
		assert.Nil(t, kv)

		kv, _ = fs.Get(context.Background(), store.DomainKey("example.com", "www"))
		assert.NotNil(t, kv)

		kv, _ = fs.Get(context.Background(), store.SkydnsRecordKey("/skydns", "example.com", "www", "0001"))
		assert.NotNil(t, kv)
	})
}

func TestRecordCreate(t *testing.T) {
	t.Run("A_record", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		r := &store.Record{Type: "A", Value: "1.2.3.4", TTL: 300}
		result, err := svc.Record.Create(context.Background(), "example.com", "www", r)
		require.NoError(t, err)
		assert.Equal(t, "0001", result.ID)
		assert.Equal(t, "A", result.Type)

		d, _ := fs.GetDomain(context.Background(), "example.com", "www")
		assert.Equal(t, 1, d.RecordCount)
		assert.Equal(t, 1, d.LastRecordID)
	})

	t.Run("duplicate", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		r := &store.Record{Type: "A", Value: "1.2.3.4", TTL: 300}
		_, err := svc.Record.Create(context.Background(), "example.com", "www", r)
		require.NoError(t, err)

		r2 := &store.Record{Type: "A", Value: "1.2.3.4", TTL: 300}
		_, err = svc.Record.Create(context.Background(), "example.com", "www", r2)
		assert.ErrorIs(t, err, apperr.RecordExists)
	})

	t.Run("invalid_type", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		r := &store.Record{Type: "CNAME", Value: "x", TTL: 300}
		_, err := svc.Record.Create(context.Background(), "example.com", "www", r)
		assert.ErrorIs(t, err, apperr.RecordTypeInvalid)
	})

	t.Run("srv_record", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		prio := 10
		port := 8080
		r := &store.Record{Type: "SRV", Value: "srv.example.com.", TTL: 300, Priority: &prio, Port: &port}
		result, err := svc.Record.Create(context.Background(), "example.com", "www", r)
		require.NoError(t, err)
		assert.Equal(t, "0001", result.ID)
		require.NotNil(t, result.Weight)
		assert.Equal(t, 0, *result.Weight)
	})
}

func TestRecordUpdate(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

	t.Run("update_value", func(t *testing.T) {
		newVal := "5.6.7.8"
		result, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "5.6.7.8", result.Value)
	})

	t.Run("update_no_fields", func(t *testing.T) {
		_, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", nil, nil, nil, nil, nil)
		assert.ErrorIs(t, err, apperr.Validation)
	})

	t.Run("non_srv_with_priority", func(t *testing.T) {
		prio := 10
		_, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", nil, nil, &prio, nil, nil)
		assert.ErrorIs(t, err, apperr.Validation)
	})

	t.Run("srv_out_of_range_priority", func(t *testing.T) {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		prio := 10
		port := 8080
		r := &store.Record{Type: "SRV", Value: "srv.example.com.", TTL: 300, Priority: &prio, Port: &port}
		_, err := svc.Record.Create(context.Background(), "example.com", "www", r)
		require.NoError(t, err)

		badPrio := 99999
		_, err = svc.Record.Update(context.Background(), "example.com", "www", "0001", nil, nil, &badPrio, nil, nil)
		assert.ErrorIs(t, err, apperr.Validation)
	})
}

func TestRecordCreateZoneNotFound(t *testing.T) {
	svc, fs := newTestServices()
	seedDomain(fs, "example.com", "www")
	r := &store.Record{Type: "A", Value: "1.2.3.4", TTL: 300}
	_, err := svc.Record.Create(context.Background(), "example.com", "www", r)
	assert.ErrorIs(t, err, apperr.ZoneNotFound)
}

func TestRecordDeleteZoneNotFound(t *testing.T) {
	svc, fs := newTestServices()
	seedDomain(fs, "example.com", "www")
	err := svc.Record.Delete(context.Background(), "example.com", "www", "0001")
	assert.ErrorIs(t, err, apperr.ZoneNotFound)
}

func TestRecordDelete(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

	err := svc.Record.Delete(context.Background(), "example.com", "www", "0001")
	require.NoError(t, err)

	kv, _ := fs.Get(context.Background(), store.RecordKey("example.com", "www", "0001"))
	assert.Nil(t, kv)

	d, _ := fs.GetDomain(context.Background(), "example.com", "www")
	assert.Equal(t, 0, d.RecordCount)
	assert.Equal(t, 1, d.LastRecordID)
}

func seedZone(fs *fakeStore, zone string) {
	ts := time.Now().UTC().Format(time.RFC3339)
	z := &store.Zone{Zone: zone, DomainCount: 0, CreatedAt: ts, UpdatedAt: ts}
	data, _ := store.MarshalZone(z)
	fs.Put(context.Background(), store.ZoneKey(zone), data)
}

func seedDomain(fs *fakeStore, zone, domain string) {
	ts := time.Now().UTC().Format(time.RFC3339)
	name := domain + "." + zone
	if domain == "@" {
		name = zone
	}
	d := &store.Domain{
		Zone: zone, Domain: domain, Name: name,
		RecordCount: 0, LastRecordID: 0,
		CreatedAt: ts, UpdatedAt: ts,
	}
	data, _ := store.MarshalDomain(d)
	fs.Put(context.Background(), store.DomainKey(zone, domain), data)
}

func seedRecord(fs *fakeStore, zone, domain, id, rtype, value string) {
	r := &store.Record{ID: id, Type: rtype, Value: value, TTL: 300}
	data, _ := store.MarshalRecord(r)
	fs.Put(context.Background(), store.RecordKey(zone, domain, id), data)

	d, _ := fs.GetDomain(context.Background(), zone, domain)
	if d != nil {
		d.RecordCount++
		d.LastRecordID = max(d.LastRecordID, parseInt(id))
		d.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		dData, _ := store.MarshalDomain(d)
		fs.Put(context.Background(), store.DomainKey(zone, domain), dData)
	}

	prefix := fs.SkydnsPrefix()
	msg := store.BuildSkydnsMsg(r)
	msgData, _ := store.MarshalSkydnsMsg(&msg)
	fs.Put(context.Background(), store.SkydnsRecordKey(prefix, zone, domain, id), msgData)
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// conflictStore wraps a Store and forces the first TxnCAS to observe a
// competing write (via hook), simulating a concurrent committer that lands
// between the caller's read and CAS. The CAS mismatch follows naturally from
// the bumped ModRevision; subsequent TxnCAS calls pass through unchanged.
type conflictStore struct {
	store.Store
	once sync.Once
	hook func(ctx context.Context, s store.Store)
}

func (c *conflictStore) TxnCAS(ctx context.Context, key string, modRevision int64, ops []store.Op) (bool, error) {
	c.once.Do(func() {
		if c.hook != nil {
			c.hook(ctx, c.Store)
		}
	})
	return c.Store.TxnCAS(ctx, key, modRevision, ops)
}

// TestRecordUpdateReMergeOnCASConflict verifies that on a CAS conflict the
// Update retry re-reads the record and re-merges its change on top of the
// competing write, so neither change is lost (the core fix for #1).
func TestRecordUpdateReMergeOnCASConflict(t *testing.T) {
	ctx := context.Background()
	fs := newFakeStore()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

	// Competing writer changes TTL 300 -> 600, leaving Value unchanged.
	rival := &store.Record{ID: "0001", Type: "A", Value: "1.2.3.4", TTL: 600}
	rivalData, _ := store.MarshalRecord(rival)
	rivalKey := store.RecordKey("example.com", "www", "0001")

	conflict := &conflictStore{
		Store: fs,
		hook: func(ctx context.Context, s store.Store) {
			_ = s.Put(ctx, rivalKey, rivalData)
		},
	}
	jwtMgr := jwt.NewManager("test-secret", 30*time.Minute, 168*time.Hour)
	svc := service.NewServices(conflict, jwtMgr)

	newVal := "5.6.7.8"
	result, err := svc.Record.Update(ctx, "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", result.Value, "update value must be applied")
	assert.Equal(t, 600, result.TTL, "concurrent TTL change must not be overwritten")

	got, _ := fs.GetRecord(ctx, "example.com", "www", "0001")
	assert.Equal(t, "5.6.7.8", got.Value)
	assert.Equal(t, 600, got.TTL)
}

// TestRecordUpdateConcurrentNoLostUpdate runs two concurrent Updates on the
// same record targeting different fields. After both complete, both changes
// must be present (no silent lost update). Run with -race.
func TestRecordUpdateConcurrentNoLostUpdate(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		svc, fs := newTestServices()
		seedZone(fs, "example.com")
		seedDomain(fs, "example.com", "www")
		seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

		var wg sync.WaitGroup
		wg.Add(2)
		barrier := make(chan struct{})
		var err1, err2 error
		go func() {
			defer wg.Done()
			<-barrier
			v := "5.6.7.8"
			_, err1 = svc.Record.Update(ctx, "example.com", "www", "0001", &v, nil, nil, nil, nil)
		}()
		go func() {
			defer wg.Done()
			<-barrier
			ttl := 600
			_, err2 = svc.Record.Update(ctx, "example.com", "www", "0001", nil, &ttl, nil, nil, nil)
		}()
		close(barrier)
		wg.Wait()

		require.NoError(t, err1, "iter %d", i)
		require.NoError(t, err2, "iter %d", i)

		got, err := fs.GetRecord(ctx, "example.com", "www", "0001")
		require.NoError(t, err, "iter %d", i)
		assert.Equal(t, "5.6.7.8", got.Value, "iter %d: value lost (concurrent update)", i)
		assert.Equal(t, 600, got.TTL, "iter %d: ttl lost (concurrent update)", i)
	}
}

func TestRecordUpdateCASRetry(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")
	fs.casFailCount = 2 // first two CAS attempts conflict; 3rd succeeds

	newVal := "5.6.7.8"
	result, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", result.Value)
}

func TestRecordUpdateCASRetryExhausted(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")
	fs.casFailCount = 3 // all retries conflict

	newVal := "5.6.7.8"
	_, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
	assert.ErrorIs(t, err, apperr.InternalError)
}

func TestRecordUpdateStoreGetError(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")
	fs.getErr = errors.New("store down")

	newVal := "5.6.7.8"
	_, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store down")
}

func TestRecordUpdateTxnCASError(t *testing.T) {
	svc, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")
	fs.txnCASErr = errors.New("etcd txn failed")

	newVal := "5.6.7.8"
	_, err := svc.Record.Update(context.Background(), "example.com", "www", "0001", &newVal, nil, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd txn failed")
}

// TestZoneCreateConcurrent runs two concurrent Creates of the same zone.
// Exactly one must succeed; the other must get ZoneExists. Run with -race.
func TestZoneCreateConcurrent(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		svc, _ := newTestServices()
		var wg sync.WaitGroup
		wg.Add(2)
		barrier := make(chan struct{})
		var err1, err2 error
		go func() {
			defer wg.Done()
			<-barrier
			_, err1 = svc.Zone.Create(ctx, "example.com")
		}()
		go func() {
			defer wg.Done()
			<-barrier
			_, err2 = svc.Zone.Create(ctx, "example.com")
		}()
		close(barrier)
		wg.Wait()

		ok1 := err1 == nil
		ok2 := err2 == nil
		assert.True(t, ok1 != ok2, "iter %d: expected exactly one success, got err1=%v err2=%v", i, err1, err2)
		if !ok1 {
			assert.ErrorIs(t, err1, apperr.ZoneExists, "iter %d", i)
		}
		if !ok2 {
			assert.ErrorIs(t, err2, apperr.ZoneExists, "iter %d", i)
		}
	}
}

func TestZoneCreateCASRetry(t *testing.T) {
	svc, fs := newTestServices()
	fs.casFailCount = 2 // first two CAS attempts conflict; 3rd succeeds

	z, err := svc.Zone.Create(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", z.Zone)
}

func TestZoneCreateCASRetryExhausted(t *testing.T) {
	svc, fs := newTestServices()
	fs.casFailCount = 3 // all retries conflict

	_, err := svc.Zone.Create(context.Background(), "example.com")
	assert.ErrorIs(t, err, apperr.InternalError)
}

func TestZoneCreateStoreGetError(t *testing.T) {
	svc, fs := newTestServices()
	fs.getErr = errors.New("store down")

	_, err := svc.Zone.Create(context.Background(), "example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store down")
}
