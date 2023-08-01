package newxxmp

import (
	"strings"
)

type Attributes []Attribute

func (a *Attributes) AddAttr(k, v string) {
	*a = append(*a, NewAttribute(k, v))
}

func NewAttribute(k, v string) Attribute {
	//log.Println("create key:", k, "value:", v)
	return Attribute{key: k, value: v}
}

type Attribute struct {
	key   string
	value string
}

func (a *Attribute) Value() string {
	return a.value
}

func (a Attribute) Key() string {
	return a.key
}

func (a *Attribute) EqualKey(k string) bool {
	return strings.Compare(a.key, k) == 0
}
