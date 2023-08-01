package ixxmp

type IToken interface {
	GetTokenByte() byte
	GetTokenString() string
	GetTokenBytes() []byte
}

type ITokenList interface {
	AddItem(token IToken)
	GetItems() []IToken
	GetBytes(headBit ...bool) []byte
}
