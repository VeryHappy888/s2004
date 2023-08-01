package waver

// WAVInterface Wa ver
type WAVInterface interface {
	GetWAVer() string
	GetWADictionary() []string
	GetSecondaryDictionary() []string
	GetNoiseProtocolVersion() []byte
}
