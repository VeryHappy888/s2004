package node

import "strings"

func NewJid(s string) JId {
	return JId{S: s}
}

type JId struct {
	S string
}

func (j *JId) GetRawString() string {
	return ""
}

func (j JId) Jid() string {
	if !strings.Contains(j.S, "@s.whatsapp.net") {
		j.S = j.S + "@s.whatsapp.net"
	}
	return j.S
}

func (j *JId) RawId() string {
	if strings.Contains(j.S, "@") {
		j.S = strings.Split(j.S, "@")[0]
	}
	return j.S
}

func (j *JId) GroupId() string {
	if j.S == "status" || j.S == "status@broadcast" {
		j.S = "status@broadcast"
		return j.S
	} else if !strings.Contains(j.S, "@g.us") {
		j.S = j.S + "@g.us"
	}
	return j.S
}
