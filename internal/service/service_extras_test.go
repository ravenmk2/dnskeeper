package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/service"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AuthService.Refresh ---

func TestAuthRefreshSuccess(t *testing.T) {
	fs := newFakeStore()
	jwtMgr := jwt.NewManager("test-secret", 30*time.Minute, 168*time.Hour)
	svcs := service.NewServices(fs, jwtMgr)
	seedUser(t, fs, "alice", "Pass1234", "admin", false)

	_, refresh, err := svcs.Auth.Login(context.Background(), "alice", "Pass1234")
	require.NoError(t, err)

	newAccess, newRefresh, err := svcs.Auth.Refresh(context.Background(), refresh)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	// access 与 refresh 的 audience 不同,token 必不同
	assert.NotEqual(t, newAccess, newRefresh)

	// 验证返回的 token 是有效的 access/refresh token,且 claims 携带正确用户
	accessClaims, err := jwtMgr.ParseAccess(newAccess)
	require.NoError(t, err)
	assert.Equal(t, "alice", accessClaims.UserID)
	assert.Equal(t, "admin", accessClaims.UserType)

	refreshClaims, err := jwtMgr.ParseRefresh(newRefresh)
	require.NoError(t, err)
	assert.Equal(t, "alice", refreshClaims.UserID)
}

func TestAuthRefreshInvalidToken(t *testing.T) {
	svcs, _ := newTestServices()
	_, _, err := svcs.Auth.Refresh(context.Background(), "not-a-valid-token")
	assert.Error(t, err)
	assert.Equal(t, apperr.InvalidToken, err)
}

func TestAuthRefreshUserDeleted(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)
	_, refresh, err := svcs.Auth.Login(context.Background(), "alice", "Pass1234")
	require.NoError(t, err)

	fs.Delete(context.Background(), store.UserKey("alice"))

	_, _, err = svcs.Auth.Refresh(context.Background(), refresh)
	assert.Error(t, err)
	assert.Equal(t, apperr.InvalidToken, err)
}

// --- AuthService.GetUser ---

func TestAuthGetUser(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "admin", false)

	u, err := svcs.Auth.GetUser(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.ID)
	assert.Equal(t, "admin", u.UserType)

	_, err = svcs.Auth.GetUser(context.Background(), "nobody")
	assert.Error(t, err)
	assert.Equal(t, apperr.UserNotFound, err)
}

// --- UserService.List ---

func TestUserList(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "admin", false)
	seedUser(t, fs, "bob", "Pass1234", "normal", false)
	users, err := svcs.User.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

// --- UserService.Update (normal paths) ---

func TestUserUpdatePassword(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Old1234", "normal", false)

	u, err := svcs.User.Update(context.Background(), "alice", "New1234", "")
	require.NoError(t, err)
	assert.Equal(t, "normal", u.UserType)

	_, _, err = svcs.Auth.Login(context.Background(), "alice", "New1234")
	require.NoError(t, err)
	_, _, err = svcs.Auth.Login(context.Background(), "alice", "Old1234")
	assert.Error(t, err)
}

func TestUserUpdateUserType(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	u, err := svcs.User.Update(context.Background(), "alice", "", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.UserType)
}

func TestUserUpdateBuiltinPassword(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "admin", "Pass1234", "admin", true)

	u, err := svcs.User.Update(context.Background(), "admin", "New1234", "")
	require.NoError(t, err)
	assert.True(t, u.Builtin)
	assert.Equal(t, "admin", u.UserType)
}

func TestUserUpdateBothFields(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	u, err := svcs.User.Update(context.Background(), "alice", "New1234", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.UserType)
	_, _, err = svcs.Auth.Login(context.Background(), "alice", "New1234")
	require.NoError(t, err)
}

func TestUserUpdateEmpty(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	_, err := svcs.User.Update(context.Background(), "alice", "", "")
	assert.Equal(t, apperr.Validation, err)
}

func TestUserUpdateNotFound(t *testing.T) {
	svcs, _ := newTestServices()
	_, err := svcs.User.Update(context.Background(), "nobody", "New1234", "")
	assert.Equal(t, apperr.UserNotFound, err)
}

func TestUserUpdateInvalidUserType(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	_, err := svcs.User.Update(context.Background(), "alice", "", "superadmin")
	assert.Equal(t, apperr.Validation, err)
}

func TestUserUpdateWeakPassword(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	_, err := svcs.User.Update(context.Background(), "alice", "weak", "")
	assert.Equal(t, apperr.WeakPassword, err)
}

// --- UserService.Delete (normal) ---

func TestUserDeleteNormal(t *testing.T) {
	svcs, fs := newTestServices()
	seedUser(t, fs, "alice", "Pass1234", "normal", false)

	require.NoError(t, svcs.User.Delete(context.Background(), "alice"))
	_, err := svcs.Auth.GetUser(context.Background(), "alice")
	assert.Equal(t, apperr.UserNotFound, err)
}

func TestUserDeleteNotFound(t *testing.T) {
	svcs, _ := newTestServices()
	err := svcs.User.Delete(context.Background(), "nobody")
	assert.Equal(t, apperr.UserNotFound, err)
}

// --- ZoneService.List ---

func TestZoneList(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedZone(fs, "test.org")
	zones, err := svcs.Zone.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, zones, 2)
}

// --- ZoneService.Update ---

func TestZoneUpdateSuccess(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")

	z, err := svcs.Zone.Update(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", z.Zone)

	stored, err := fs.GetZone(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, z.UpdatedAt, stored.UpdatedAt)
}

func TestZoneUpdateNotFound(t *testing.T) {
	svcs, _ := newTestServices()
	_, err := svcs.Zone.Update(context.Background(), "nope.example.com")
	assert.Equal(t, apperr.ZoneNotFound, err)
}

func TestZoneUpdateCASRetry(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	fs.casFailCount = 2

	_, err := svcs.Zone.Update(context.Background(), "example.com")
	require.NoError(t, err)
}

func TestZoneUpdateCASExhausted(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	fs.casFailCount = 3

	_, err := svcs.Zone.Update(context.Background(), "example.com")
	assert.Equal(t, apperr.InternalError, err)
}

// --- DomainService.List ---

func TestDomainListSuccess(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedDomain(fs, "example.com", "api")
	domains, err := svcs.Domain.List(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Len(t, domains, 2)
}

func TestDomainListZoneNotFound(t *testing.T) {
	svcs, _ := newTestServices()
	_, err := svcs.Domain.List(context.Background(), "nope.example.com")
	assert.Equal(t, apperr.ZoneNotFound, err)
}

// --- DomainService.Update ---

func TestDomainUpdateSuccess(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")

	d, err := svcs.Domain.Update(context.Background(), "example.com", "www")
	require.NoError(t, err)
	assert.Equal(t, "www", d.Domain)

	stored, err := fs.GetDomain(context.Background(), "example.com", "www")
	require.NoError(t, err)
	assert.Equal(t, d.UpdatedAt, stored.UpdatedAt)
}

func TestDomainUpdateNotFound(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")

	_, err := svcs.Domain.Update(context.Background(), "example.com", "www")
	assert.Equal(t, apperr.DomainNotFound, err)
}

func TestDomainUpdateCASRetry(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	fs.casFailCount = 2

	_, err := svcs.Domain.Update(context.Background(), "example.com", "www")
	require.NoError(t, err)
}

func TestDomainUpdateCASExhausted(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	fs.casFailCount = 3

	_, err := svcs.Domain.Update(context.Background(), "example.com", "www")
	assert.Equal(t, apperr.InternalError, err)
}

// --- RecordService.List ---

func TestRecordListSuccess(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")
	seedRecord(fs, "example.com", "www", "0002", "A", "5.6.7.8")
	records, err := svcs.Record.List(context.Background(), "example.com", "www")
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestRecordListZoneNotFound(t *testing.T) {
	svcs, _ := newTestServices()
	_, err := svcs.Record.List(context.Background(), "nope.example.com", "www")
	assert.Equal(t, apperr.ZoneNotFound, err)
}

func TestRecordListDomainNotFound(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	_, err := svcs.Record.List(context.Background(), "example.com", "www")
	assert.Equal(t, apperr.DomainNotFound, err)
}

// --- RecordService.Get ---

func TestRecordGetSuccess(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	seedRecord(fs, "example.com", "www", "0001", "A", "1.2.3.4")

	r, err := svcs.Record.Get(context.Background(), "example.com", "www", "0001")
	require.NoError(t, err)
	assert.Equal(t, "A", r.Type)
	assert.Equal(t, "1.2.3.4", r.Value)
}

func TestRecordGetInvalidID(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")

	_, err := svcs.Record.Get(context.Background(), "example.com", "www", "abc")
	assert.Equal(t, apperr.Validation, err)
}

func TestRecordGetNotFound(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")

	_, err := svcs.Record.Get(context.Background(), "example.com", "www", "9999")
	assert.Equal(t, apperr.RecordNotFound, err)
}

// --- RecordService.Create: type coverage ---

func TestRecordCreateAAAA(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")

	r, err := svcs.Record.Create(context.Background(), "example.com", "www", &store.Record{
		Type: "AAAA", Value: "::1", TTL: 300,
	})
	require.NoError(t, err)
	assert.Equal(t, "0001", r.ID)

	prefix := fs.SkydnsPrefix()
	kv, err := fs.Get(context.Background(), store.SkydnsRecordKey(prefix, "example.com", "www", "0001"))
	require.NoError(t, err)
	require.NotNil(t, kv)
	assert.Contains(t, string(kv.Value), "::1")
	assert.Contains(t, string(kv.Value), "host")
}

func TestRecordCreateTXT(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")

	r, err := svcs.Record.Create(context.Background(), "example.com", "www", &store.Record{
		Type: "TXT", Value: "v=spf1", TTL: 300,
	})
	require.NoError(t, err)
	assert.Equal(t, "0001", r.ID)

	prefix := fs.SkydnsPrefix()
	kv, _ := fs.Get(context.Background(), store.SkydnsRecordKey(prefix, "example.com", "www", "0001"))
	require.NotNil(t, kv)
	assert.Contains(t, string(kv.Value), "v=spf1")
	assert.Contains(t, string(kv.Value), "text")
}

func TestRecordCreateSRVDefaultWeight(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	pri, port := 10, 5060

	r, err := svcs.Record.Create(context.Background(), "example.com", "www", &store.Record{
		Type: "SRV", Value: "srv.example.com", TTL: 300, Priority: &pri, Port: &port,
	})
	require.NoError(t, err)
	assert.Equal(t, "0001", r.ID)
	require.NotNil(t, r.Weight)
	assert.Equal(t, 0, *r.Weight)
}

// --- SRV record duplicate detection (service.go:159-162) ---

func TestSRVRecordDuplicate(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	pri, port, weight := 10, 5060, 20
	r1 := &store.Record{Type: "SRV", Value: "srv.example.com", TTL: 300, Priority: &pri, Port: &port, Weight: &weight}
	_, err := svcs.Record.Create(context.Background(), "example.com", "www", r1)
	require.NoError(t, err)

	// 相同 SRV → RecordExists
	r2 := &store.Record{Type: "SRV", Value: "srv.example.com", TTL: 300, Priority: &pri, Port: &port, Weight: &weight}
	_, err = svcs.Record.Create(context.Background(), "example.com", "www", r2)
	assert.Equal(t, apperr.RecordExists, err)

	// 不同 port → 不重复,创建成功
	diffPort := 5061
	r3 := &store.Record{Type: "SRV", Value: "srv.example.com", TTL: 300, Priority: &pri, Port: &diffPort, Weight: &weight}
	res, err := svcs.Record.Create(context.Background(), "example.com", "www", r3)
	require.NoError(t, err)
	assert.Equal(t, "0002", res.ID)
}

// --- RecordService.Create CAS retry ---

func TestRecordCreateCASRetry(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	seedDomain(fs, "example.com", "www")
	fs.casFailCount = 2

	r, err := svcs.Record.Create(context.Background(), "example.com", "www", &store.Record{
		Type: "A", Value: "1.2.3.4", TTL: 300,
	})
	require.NoError(t, err)
	assert.Equal(t, "0001", r.ID)
}

// --- DomainService.Create CAS retry ---

func TestDomainCreateCASRetry(t *testing.T) {
	svcs, fs := newTestServices()
	seedZone(fs, "example.com")
	fs.casFailCount = 2

	d, err := svcs.Domain.Create(context.Background(), "example.com", "www")
	require.NoError(t, err)
	assert.Equal(t, "www", d.Domain)
	assert.Equal(t, "www.example.com", d.Name)
}
