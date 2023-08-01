package axolotl

import (
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"testing"
	"ws-go/libsignal/ecc"
	"ws-go/libsignal/keys/identity"
	"ws-go/libsignal/keys/prekey"
	"ws-go/libsignal/protocol"
	signalProto "ws-go/libsignal/protos"
	"ws-go/libsignal/util/bytehelper"
	"ws-go/libsignal/util/optional"
	"ws-go/protocol/axolotl/serializer"
)

func TestManager_Decrypt(t *testing.T) {
	d, _ := hex.DecodeString("126e0a1d383631373630373536373030352d3136313739343831303040672e7573124d33088bd6b91010001a2053b646afc0a7f84021c7eaa2eb78b51dc13805f47824b509315b378b72d6ee1a22210508b73958098fe1b74c7d9db02c5bc36855985ed38d699e1357d520772bce8b4e0b0b0b0b0b0b0b0b0b0b0b")
	t.Log(hex.EncodeToString(removePadding(d)))
}

func TestManager_CreateGroupSession(t *testing.T) {
	axolotlManager, _ := NewAxolotlManager("test")
	data, _ := hex.DecodeString("3308e0dab8fa0210011a20dfb19187b6bada0861d391b9b34191c29f1a9a9fcf6a9801f38c8ad0e3394e63fd34e9e4deed4c9c92291fe2f1d300f67b31b7571d6764b2b7f7252729170f17cd5de5134b52d1c28ffde2d224caee5fe74c9d67f1b139ed501f325568eca206")
	axolotlManager.ProcessGroupSession("8617607567005-1617889232@g.us", "8617607567005@s.whatsapp.net", data)
}

func TestDecryptMsg(t *testing.T) {
	messageSerializer := serializer.ProtoPreKeySignalMessageSerializer{}
	signalMessageSerializer := serializer.ProtoSignalMessageSerializer{}
	axolotlManager, _ := NewAxolotlManager("test")
	//axolotlManager.SessionStore.LoadSession(protocol.NewSignalAddress("aaa", 0))
	//cipher := axolotlManager.getSessionCipher("aaa")

	enchex := "3308a8cedb07122105518ebe9a320e6b5814f80d5280262b65a1948393ac9aefbfb998172ef294f3411a2105e5242c5dbc1f83bc7c73decf7eea196c3c98fd43b176a80494dd421e0f9df1092252330a2105aee77b615158d8fee2b10be79061f722d77158615ab0ef34579a5c2c81b642691000180022209be85819845f46592e4880754c982287a1771fe53dbda88d995b86368af9dd5e30c333864ceb6f53288de09dfd043000"
	enc, _ := hex.DecodeString(enchex)
	prekeymessage, _ := protocol.NewPreKeySignalMessageFromBytes(enc, &messageSerializer, &signalMessageSerializer)

	axolotlManager.getSessionBuilder("8617607567005").Process(prekeymessage)
	cipher := axolotlManager.getSessionCipher("8617607567005")
	d, err := cipher.Decrypt(prekeymessage.WhisperMessage())

	t.Log(err, hex.EncodeToString(d))
}

func TestQT(t *testing.T) {
	dataHex := "0ad30208031221052367ac4923657187e382906d0b203449b650e264f2a08002aa9f514492adae591a2105282c60808cc705d43726c366e3a76c3f4d2ecad5012588436e87d27cdc954e482220c84f43640fefa48fd22950a94faf805f892f2e3ea64cf3f8a5f4e05e59863db62800326b0a2105941a7e5491a8d3f38b616d7571c4548d836852af89fe5e16882921b72988367d122030f1b948e39aa7953bda11ebadd531366074ec6b63d2de6fe9bcedf397bf9f491a2408001220c58a611fb8a22d7da809949bb3dead918f7d3156744a744e1b42b074d713e9b63a490a2105c1116de188d23e15ea7b275ff412c8e6bcbf764bff90cc1610ea00748299be651a240801122036e006a5b9c941754e5f61ad53f3b561614043413e131c6a8cfa241d033055e950c1a1c68202589fb1b294066a2105ee9087ba4b18c672b85d777e8af2896ba99037f401597a70003056cef7fbc24a12fd0208031221052367ac4923657187e382906d0b203449b650e264f2a08002aa9f514492adae591a2105282c60808cc705d43726c366e3a76c3f4d2ecad5012588436e87d27cdc954e482220ba1447a107edbdc0818d8cdaec6474001ba763caed2844baac0201536c314113326b0a2105494be91967187f997ffcaf7082db6102feae0e4c2d60c5018165a3b9e5f85f55122038d916c4163bbc310e43cbc01df01cc86e44a55c85eafe97c4d41a0d70ce69621a2408031220b81cbcbd660d378a4e7cd046b70e73cf63f20c30cf2c84aeca77cf3bb57ec3ca3a490a2105e565340dc3f0b3d8fdc677adfc45fcce4583ee79ad4d58d4ff1f1ce702ed22721a2408001220b1d58b7bcf618441d3f57a2f4fabb162e790dceb2acb5a4911c516603c2dadab4a2a08371221055876325106f44c8f12ca7c73b0eba5cfc2b2144a97a189c5368663497942475618d68ea20350c1a1c68202589fb1b294066a21055876325106f44c8f12ca7c73b0eba5cfc2b2144a97a189c536866349794247561200"
	data, err := hex.DecodeString(dataHex)
	if err != nil {
		t.Fatal(err)
	}

	messageStructure := signalProto.PreKeySignalMessage{}
	err = proto.Unmarshal(data, &messageStructure)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(hex.EncodeToString(messageStructure.Message))
}

func TestCreateSession(t *testing.T) {
	axolotlManager, _ := NewAxolotlManager("test")
	/*registrationID uint32,
	deviceID uint32,
	preKeyID *optional.Uint32,
	signedPreKeyID uint32,
	preKeyPublic ecc.ECPublicKeyable,
	signedPreKeyPublic ecc.ECPublicKeyable,
	signedPreKeySig [64]byte,
	identityKey *identity.Key*/
	preKeyPublicHex := "058f98bdd04315becae3322dce6cb880665b04c0502a3984964709bcbb5c6a8976"
	preKeyPublic, err := hex.DecodeString(preKeyPublicHex)
	if err != nil {
		t.Fatal(err)
	}

	preKeyPublicKey, err := ecc.DecodePoint(preKeyPublic, 0)
	if err != nil {
		t.Fatal(err)
	}

	signedPreKeyPublicHex := "05e565340dc3f0b3d8fdc677adfc45fcce4583ee79ad4d58d4ff1f1ce702ed2272"
	signedPreKeyPublic, err := hex.DecodeString(signedPreKeyPublicHex)
	if err != nil {
		t.Fatal(err)
	}

	signedPreKey, err := ecc.DecodePoint(signedPreKeyPublic, 0)
	if err != nil {
		t.Fatal(err)
	}

	signedPreKeySigHex := "1b1d44bdb7fc57394ba33ebf39096646e52e1c9c8c5bd0f2f802bbc326c54c521bf0e4578ded7a6970b61c1ecf219e4572267b140917ae236536e15b9d737e84"
	signedPreKeySig, err := hex.DecodeString(signedPreKeySigHex)
	if err != nil {
		t.Fatal(err)
	}

	identityHex := "05282c60808cc705d43726c366e3a76c3f4d2ecad5012588436e87d27cdc954e48"
	identityData, err := hex.DecodeString(identityHex)
	if err != nil {
		t.Fatal(err)
	}

	identityKey, err := ecc.DecodePoint(identityData, 0)
	if err != nil {
		t.Fatal(err)
	}

	bundle := prekey.NewBundle(
		2095653633,
		0,
		optional.NewOptionalUint32(10546496),
		18,
		preKeyPublicKey,
		signedPreKey,
		bytehelper.SliceToArray64(signedPreKeySig),
		identity.NewKey(identityKey),
	)
	id := "8618665179087"
	ContextHex := "0a01510c0c0c0c0c0c0c0c0c0c0c0c"
	Context, err := hex.DecodeString(ContextHex)
	if err != nil {
		t.Fatal(err)
	}

	axolotlManager.CreateSession(id, bundle)
	encrypeData, err := axolotlManager.Encrypt(id, Context, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(hex.EncodeToString(encrypeData.Serialize()), encrypeData.Type())
}
