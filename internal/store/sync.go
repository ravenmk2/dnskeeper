package store

import (
	"encoding/json"
	"strings"
)

type SkydnsMsg struct {
	Host     string `json:"host,omitempty"`
	Text     string `json:"text,omitempty"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
}

func fqdn(zone, domain string) string {
	if domain == "@" {
		return zone
	}
	return domain + "." + zone
}

func reverseLabels(name string) string {
	labels := strings.Split(name, ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, "/")
}

func SkydnsRecordKey(prefix, zone, domain, recordID string) string {
	return prefix + "/" + reverseLabels(fqdn(zone, domain)) + "/" + recordID
}

func SkydnsDomainPrefix(prefix, zone, domain string) string {
	return prefix + "/" + reverseLabels(fqdn(zone, domain)) + "/"
}

func SkydnsZonePrefix(prefix, zone string) string {
	return prefix + "/" + reverseLabels(zone) + "/"
}

func BuildSkydnsMsg(r *Record) SkydnsMsg {
	msg := SkydnsMsg{TTL: r.TTL}
	switch r.Type {
	case "A", "AAAA":
		msg.Host = r.Value
	case "TXT":
		msg.Text = r.Value
	case "SRV":
		msg.Host = r.Value
		msg.Priority = r.Priority
		msg.Port = r.Port
		msg.Weight = r.Weight
	}
	return msg
}

func MarshalSkydnsMsg(msg *SkydnsMsg) ([]byte, error) {
	return json.Marshal(msg)
}
