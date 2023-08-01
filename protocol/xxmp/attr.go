package xxmp

func NewAttribute(k, v string) Attribute {
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
