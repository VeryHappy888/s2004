package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"sort"
	"strings"
)

// CalcPHash
func CalcPHash(jids []string) string {
	for j, jid := range jids {
		if !strings.Contains(jid, "@s.whatsapp.net") {
			jid += "@s.whatsapp.net"
		}
		if !strings.Contains(jid, ".0:0") {
			i := strings.Index(jid, "@")
			jids[j] = jid[:i] + ".0:0" + jid[i:]
		}
	}
	sort.Strings(jids)
	hash := sha256.New()
	for _, jid := range jids {
		hash.Write([]byte(jid))
	}
	return "2:" + base64.StdEncoding.EncodeToString(hash.Sum(nil)[:6])
}
