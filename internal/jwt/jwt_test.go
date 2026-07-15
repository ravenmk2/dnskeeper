package jwt_test

import (
	"strings"
	"testing"
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMgr() *jwt.Manager {
	return jwt.NewManager("test-secret", time.Minute, time.Hour)
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	mgr := newMgr()
	tok, err := mgr.Issue("u-1", "alice", "admin", jwt.ClaimTypeAccess)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := mgr.Parse(tok)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, "u-1", claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, "admin", claims.UserType)
	assert.Equal(t, "u-1", claims.Subject)
	require.Len(t, claims.Audience, 1)
	assert.Equal(t, jwt.ClaimTypeAccess, claims.Audience[0])
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(claims.IssuedAt.Time))
}

func TestIssueRefreshUsesRefreshTTL(t *testing.T) {
	mgr := jwt.NewManager("s", time.Minute, time.Hour)
	access, err := mgr.Issue("u", "n", "u", jwt.ClaimTypeAccess)
	require.NoError(t, err)
	refresh, err := mgr.Issue("u", "n", "u", jwt.ClaimTypeRefresh)
	require.NoError(t, err)

	ac, err := mgr.Parse(access)
	require.NoError(t, err)
	rc, err := mgr.Parse(refresh)
	require.NoError(t, err)
	assert.True(t, rc.ExpiresAt.Time.After(ac.ExpiresAt.Time))
}

func TestIssuePair(t *testing.T) {
	mgr := newMgr()
	access, refresh, err := mgr.IssuePair("u-1", "alice", "admin")
	require.NoError(t, err)
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)
	assert.NotEqual(t, access, refresh)

	_, err = mgr.ParseAccess(access)
	require.NoError(t, err)
	_, err = mgr.ParseRefresh(access)
	assert.Error(t, err)

	_, err = mgr.ParseRefresh(refresh)
	require.NoError(t, err)
	_, err = mgr.ParseAccess(refresh)
	assert.Error(t, err)
}

func TestParseWrongSecret(t *testing.T) {
	signer := jwt.NewManager("secret-A", time.Minute, time.Hour)
	verifier := jwt.NewManager("secret-B", time.Minute, time.Hour)
	tok, err := signer.Issue("u", "n", "u", jwt.ClaimTypeAccess)
	require.NoError(t, err)
	_, err = verifier.Parse(tok)
	assert.Error(t, err)
}

// TestParseAlgConfusion 验证 jwt.go:74 的签名方法强制校验。
// 手造 alg=none 的 token,Parse 必须拒绝。
func TestParseAlgConfusion(t *testing.T) {
	mgr := newMgr()
	claims := jwt.Claims{
		UserID:   "u",
		Username: "n",
		UserType: "admin",
		RegisteredClaims: gjwt.RegisteredClaims{
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   "u",
			Audience:  []string{jwt.ClaimTypeAccess},
		},
	}
	tok := gjwt.NewWithClaims(gjwt.SigningMethodNone, claims)
	tokenStr, err := tok.SignedString(gjwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = mgr.Parse(tokenStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestParseExpired(t *testing.T) {
	mgr := jwt.NewManager("s", -time.Minute, -time.Minute)
	tok, err := mgr.Issue("u", "n", "u", jwt.ClaimTypeAccess)
	require.NoError(t, err)
	_, err = mgr.Parse(tok)
	assert.Error(t, err)
}

func TestParseTampered(t *testing.T) {
	mgr := newMgr()
	tok, err := mgr.Issue("u", "n", "u", jwt.ClaimTypeAccess)
	require.NoError(t, err)
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3)
	parts[2] = "AAAA"
	tampered := strings.Join(parts, ".")
	_, err = mgr.Parse(tampered)
	assert.Error(t, err)
}

func TestParseEmptyAndMalformed(t *testing.T) {
	mgr := newMgr()
	cases := []string{"", "not-a-token", "a.b", "a.b.c", "...", "a..c"}
	for _, tc := range cases {
		_, err := mgr.Parse(tc)
		assert.Error(t, err, "token=%q", tc)
	}
}

// TestParseRefreshEmptyAudience 覆盖 jwt.go:90 的 len(Audience)==0 分支。
func TestParseRefreshEmptyAudience(t *testing.T) {
	mgr := newMgr()
	claims := jwt.Claims{
		UserID:   "u",
		Username: "n",
		UserType: "u",
		RegisteredClaims: gjwt.RegisteredClaims{
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   "u",
		},
	}
	tok := gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims)
	tokenStr, err := tok.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	_, err = mgr.ParseRefresh(tokenStr)
	assert.Error(t, err)
	_, err = mgr.ParseAccess(tokenStr)
	assert.Error(t, err)
}
