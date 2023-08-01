package protocol

type CiphertextMessage interface {
	Serialize() []byte
	Type() uint32
}

const UnsupportedVersion = 1
const CurrentVersion = 3

const WHISPER_TYPE = 2
const PREKEY_TYPE = 3    // whatsapp pkmsg
const SENDERKEY_TYPE = 4 //skmsg
const SENDERKEY_DISTRIBUTION_TYPE = 5

func GetEncTypeString(encType uint32) string {
	switch encType {
	case PREKEY_TYPE:
		return "pkmsg"
	case SENDERKEY_TYPE:
		return "skmsg"
	default:
		return "msg"
	}
}
