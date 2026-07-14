package service

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/store"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,30}[a-zA-Z0-9]$`)

func validateUsername(username string) bool {
	if !usernameRegex.MatchString(username) {
		return false
	}
	if strings.Contains(username, "__") || strings.Contains(username, "--") ||
		strings.Contains(username, "_-") || strings.Contains(username, "-_") {
		return false
	}
	return true
}

func passwordClasses(s string) int {
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 0x21 && c <= 0x7E:
			hasSpecial = true
		}
	}
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}
	return count
}

func validatePassword(password string) bool {
	if len(password) < 6 || len(password) > 24 {
		return false
	}
	return passwordClasses(password) >= 2
}

func validateLabel(label string) bool {
	if len(label) < 1 || len(label) > 63 {
		return false
	}
	for _, c := range label {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return isAlphaNum(label[0]) && isAlphaNum(label[len(label)-1])
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func validateZone(zone string) bool {
	if len(zone) < 1 || len(zone) > 253 {
		return false
	}
	labels := strings.Split(zone, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if !validateLabel(label) {
			return false
		}
	}
	return true
}

func validateDomain(domain string) bool {
	if domain == "@" {
		return true
	}
	if len(domain) < 1 || len(domain) > 253 {
		return false
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if !validateLabel(label) {
			return false
		}
	}
	return true
}

func validateRecordType(t string) bool {
	return t == "A" || t == "AAAA" || t == "SRV" || t == "TXT"
}

func validateRecordValue(recordType, value string) bool {
	switch recordType {
	case "A":
		ip := net.ParseIP(value)
		return ip != nil && ip.To4() != nil
	case "AAAA":
		ip := net.ParseIP(value)
		return ip != nil && ip.To4() == nil
	case "SRV":
		return len(value) > 0
	case "TXT":
		return len(value) <= 255
	}
	return false
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func formatRecordID(n int) string {
	return fmt.Sprintf("%04d", n)
}

func intPtrEq(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func recordsDuplicate(a, b *store.Record) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case "A", "AAAA", "TXT":
		return a.Value == b.Value
	case "SRV":
		return a.Value == b.Value &&
			intPtrEq(a.Priority, b.Priority) &&
			intPtrEq(a.Port, b.Port) &&
			intPtrEq(a.Weight, b.Weight)
	}
	return false
}

type Services struct {
	Auth   *AuthService
	User   *UserService
	Zone   *ZoneService
	Domain *DomainService
	Record *RecordService
}

func NewServices(s store.Store, jwtMgr *jwt.Manager) *Services {
	return &Services{
		Auth:   &AuthService{store: s, jwt: jwtMgr},
		User:   &UserService{store: s},
		Zone:   &ZoneService{store: s},
		Domain: &DomainService{store: s},
		Record: &RecordService{store: s},
	}
}
