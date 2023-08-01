package serializer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/proto"
	"log"
	"ws-go/libsignal/groups/ratchet"
	groupRecord "ws-go/libsignal/groups/state/record"
	"ws-go/libsignal/keys/chain"
	"ws-go/libsignal/keys/message"
	"ws-go/libsignal/logger"
	"ws-go/libsignal/protocol"
	signalProto "ws-go/libsignal/protos"
	"ws-go/libsignal/serialize"
	"ws-go/libsignal/state/record"
	"ws-go/libsignal/util/optional"
)

func NewProtoSerializer() *serialize.Serializer {
	serializer := serialize.NewSerializer()
	serializer.PreKeySignalMessage = &ProtoPreKeySignalMessageSerializer{}
	serializer.SignalMessage = &ProtoSignalMessageSerializer{}
	serializer.SenderKeyDistributionMessage = &ProtoSenderKeyDistributionMessageSerializer{}
	serializer.SignedPreKeyRecord = &ProtoSignedPreKeyRecordSerializer{}
	serializer.Session = &ProtoSessionSerializer{}
	serializer.State = &ProtoStateSerializer{}
	serializer.SenderKeyMessage = &ProtoSenderKeyMessageSerializer{}
	serializer.SenderKeyRecord = &ProtoSenderKeySessionSerializer{}
	serializer.SenderKeyState = &ProtoSenderKeyStateSerializer{}
	serializer.PreKeyRecord = &ProtoPreKeyRecordSerializer{}
	return serializer
}

// JSONSenderKeySessionSerializer is a structure for serializing session records into
// and from JSON.
type ProtoSenderKeySessionSerializer struct{}

// Serialize will take a session structure and convert it to JSON bytes.
func (p *ProtoSenderKeySessionSerializer) Serialize(session *groupRecord.SenderKeyStructure) []byte {
	protoSenderKeyRecord := &signalProto.SenderKeyRecordStructure{}
	protoSenderKeyRecord.SenderKeyStates = make([]*signalProto.SenderKeyStateStructure, 0)
	for _, state := range session.SenderKeyStates {
		protoSenderKeyRecord.SenderKeyStates = append(protoSenderKeyRecord.SenderKeyStates, makeSenderKeyStateToProto(state))
	}

	senderKeyRecordData, err := proto.Marshal(protoSenderKeyRecord)
	if err != nil {
		return []byte{}
	}
	return senderKeyRecordData
}

// Deserialize will take in JSON bytes and return a session structure, which can be
// used to create a new Session Record object.
func (p *ProtoSenderKeySessionSerializer) Deserialize(serialized []byte) (*groupRecord.SenderKeyStructure, error) {
	var protoSession signalProto.SenderKeyRecordStructure
	err := proto.Unmarshal(serialized, &protoSession)
	if err != nil {
		logger.Error("Error deserializing session: ", err)
		return nil, err
	}
	senderKeyStructure := &groupRecord.SenderKeyStructure{SenderKeyStates: make([]*groupRecord.SenderKeyStateStructure, 0)}
	protoSessionSenderKeys := protoSession.GetSenderKeyStates()
	for _, key := range protoSessionSenderKeys {
		senderKeyStructure.SenderKeyStates = append(senderKeyStructure.SenderKeyStates, makeSenderKeyToStruct(key))
	}
	return senderKeyStructure, nil
}

type ProtoSenderKeyStateSerializer struct{}

// Serialize will take a session state structure and convert it to pb bytes.
func (p *ProtoSenderKeyStateSerializer) Serialize(state *groupRecord.SenderKeyStateStructure) []byte {
	protoSenderKeyState := makeSenderKeyStateToProto(state)
	senderKeyData, err := proto.Marshal(protoSenderKeyState)
	if err != nil {
		return []byte{}
	}
	return senderKeyData
}

// Deserialize will take in pb bytes and return a session state structure.
func (p *ProtoSenderKeyStateSerializer) Deserialize(serialized []byte) (*groupRecord.SenderKeyStateStructure, error) {
	var protoState signalProto.SenderKeyStateStructure
	err := proto.Unmarshal(serialized, &protoState)
	if err != nil {
		logger.Error("Error deserializing session state: ", err)
		return nil, err
	}

	return makeSenderKeyToStruct(&protoState), nil
}

type ProtoSenderKeyMessageSerializer struct{}

// Serialize will take a senderkey message and convert it to JSON bytes.
func (j *ProtoSenderKeyMessageSerializer) Serialize(message *protocol.SenderKeyMessageStructure) []byte {
	senderKeyMessage := &signalProto.SenderKeyMessage{}
	senderKeyMessage.Id = &message.ID
	senderKeyMessage.Iteration = &message.Iteration
	senderKeyMessage.Ciphertext = message.CipherText
	senderKeyMessageData, err := proto.Marshal(senderKeyMessage)
	if err != nil {
		return nil
	}
	ver := byte((message.Version<<4 | 3) & 0xFF)
	data := append([]byte{ver}, senderKeyMessageData...)
	data = append(data, message.Signature...)
	return data
}

// Deserialize will take in JSON bytes and return a message structure, which can be
// used to create a new SenderKey message object.
func (j *ProtoSenderKeyMessageSerializer) Deserialize(serialized []byte) (*protocol.SenderKeyMessageStructure, error) {
	version := (serialized[0] & 255) >> 4
	messageData := serialized[1 : len(serialized)-64]
	signature := serialized[len(serialized)-64:]
	senderKeyMessage := &signalProto.SenderKeyMessage{}
	err := proto.Unmarshal(messageData, senderKeyMessage)
	if err != nil {
		return nil, err
	}

	return &protocol.SenderKeyMessageStructure{
		ID:         senderKeyMessage.GetId(),
		Iteration:  senderKeyMessage.GetIteration(),
		CipherText: senderKeyMessage.GetCiphertext(),
		Version:    uint32(version),
		Signature:  signature,
	}, err
}

// ProtoSenderKeyDistributionMessageSerializer
type ProtoSenderKeyDistributionMessageSerializer struct{}

func (p ProtoSenderKeyDistributionMessageSerializer) Serialize(message *protocol.SenderKeyDistributionMessageStructure) []byte {
	serializeData := []byte{51} // Version
	distributionMessage := &signalProto.SenderKeyDistributionMessage{
		Id:         proto.Uint32(message.ID),
		Iteration:  proto.Uint32(message.Iteration),
		ChainKey:   message.ChainKey,
		SigningKey: message.SigningKey,
	}
	distributionMessageData, err := proto.Marshal(distributionMessage)
	if err != nil {
		return nil
	}
	serializeData = append(serializeData, distributionMessageData...)
	return serializeData
}
func (p ProtoSenderKeyDistributionMessageSerializer) Deserialize(serialized []byte) (*protocol.SenderKeyDistributionMessageStructure, error) {
	if len(serialized) <= 2 {
		return nil, errors.New("serialized len <= 2")
	}
	version := (serialized[0] & 255) >> 4
	bytes := serialized[1:]

	distributionMessage := &signalProto.SenderKeyDistributionMessage{}
	err := proto.Unmarshal(bytes, distributionMessage)
	if err != nil {
		return nil, err
	}

	return &protocol.SenderKeyDistributionMessageStructure{
		ID:         distributionMessage.GetId(),
		Iteration:  distributionMessage.GetIteration(),
		ChainKey:   distributionMessage.GetChainKey(),
		SigningKey: distributionMessage.GetSigningKey(),
		Version:    uint32(version),
	}, nil
}

//ProtoSignalMessageSerializer
type ProtoSignalMessageSerializer struct{}

// Serialize
func (p *ProtoSignalMessageSerializer) Serialize(signalMessage *protocol.SignalMessageStructure) []byte {
	//TODO 也许是对的
	ver := byte((signalMessage.Version<<4 | 3) & 0xFF)
	message := signalProto.SignalMessage{
		RatchetKey:      signalMessage.RatchetKey,
		Counter:         proto.Uint32(signalMessage.Counter),
		PreviousCounter: proto.Uint32(signalMessage.PreviousCounter),
		Ciphertext:      signalMessage.CipherText,
	}
	messageData, err := proto.Marshal(&message)
	if err != nil {
		return nil
	}
	log.Println("RatchetKey ", hex.EncodeToString(signalMessage.RatchetKey), len(signalMessage.RatchetKey))
	log.Println("messageData", hex.EncodeToString(messageData))
	d := append([]byte{ver}, messageData...)
	d = append(d, signalMessage.Mac...)

	log.Println("messageData", hex.EncodeToString(d))
	return d
}

// Deserialize
func (p *ProtoSignalMessageSerializer) Deserialize(serialized []byte) (*protocol.SignalMessageStructure, error) {
	if len(serialized) <= 2 {
		return nil, errors.New("serialized len <= 2")
	}
	log.Println("ProtoSignalMessageSerializer Deserialize :", hex.EncodeToString(serialized))
	version := (serialized[0] & 255) >> 4
	messageData := serialized[1 : len(serialized)-8]
	mac := serialized[len(serialized)-8:]

	signalMessage := &signalProto.SignalMessage{}
	err := proto.Unmarshal(messageData, signalMessage)
	if err != nil {
		return nil, err
	}
	return &protocol.SignalMessageStructure{
		RatchetKey:      signalMessage.GetRatchetKey(),
		Counter:         signalMessage.GetCounter(),
		PreviousCounter: signalMessage.GetPreviousCounter(),
		CipherText:      signalMessage.GetCiphertext(),
		Version:         int(version),
		Mac:             mac,
	}, nil
}

type ProtoPreKeySignalMessageSerializer struct{}

// Serialize will take a signal message structure and convert it to Proto bytes.
func (j *ProtoPreKeySignalMessageSerializer) Serialize(signalMessage *protocol.PreKeySignalMessageStructure) []byte {
	ver := byte((signalMessage.Version<<4 | 3) & 0xFF)
	log.Println("ProtoPreKeySignalMessageSerializer message ", hex.EncodeToString(signalMessage.Message))
	preKeySignalMessageProto := signalProto.PreKeySignalMessage{
		RegistrationId: proto.Uint32(signalMessage.RegistrationID),
		PreKeyId:       proto.Uint32(signalMessage.PreKeyID.Value),
		SignedPreKeyId: proto.Uint32(signalMessage.SignedPreKeyID),
		BaseKey:        signalMessage.BaseKey,
		IdentityKey:    signalMessage.IdentityKey,
		Message:        signalMessage.Message,
	}

	messageData, err := proto.Marshal(&preKeySignalMessageProto)
	if err != nil {
		return nil
	}
	d := append([]byte{ver}, messageData...)
	return d
}

// Deserialize will take in Proto bytes and return a signal message structure.
func (j *ProtoPreKeySignalMessageSerializer) Deserialize(serialized []byte) (*protocol.PreKeySignalMessageStructure, error) {
	if len(serialized) <= 2 {
		return nil, errors.New("serialized len <= 2")
	}
	version := (serialized[0] & 0xFF) >> 4
	data := serialized[1:]
	preKeySignalMessage := signalProto.PreKeySignalMessage{}
	// un serialize
	err := proto.Unmarshal(data, &preKeySignalMessage)
	if err != nil {
		return nil, err
	}

	return &protocol.PreKeySignalMessageStructure{
		RegistrationID: preKeySignalMessage.GetRegistrationId(),
		PreKeyID:       optional.NewOptionalUint32(preKeySignalMessage.GetPreKeyId()),
		SignedPreKeyID: preKeySignalMessage.GetSignedPreKeyId(),
		BaseKey:        preKeySignalMessage.GetBaseKey(),
		IdentityKey:    preKeySignalMessage.GetIdentityKey(),
		Message:        preKeySignalMessage.GetMessage(),
		Version:        int(version),
	}, nil
}

type ProtoSignedPreKeyRecordSerializer struct{}

// Serialize will take a signed prekey record structure and convert it to JSON bytes.
func (p *ProtoSignedPreKeyRecordSerializer) Serialize(signedPreKey *record.SignedPreKeyStructure) []byte {
	signedPreKeyRecordProto := &signalProto.SignedPreKeyRecordStructure{
		Id:         proto.Uint32(signedPreKey.ID),
		PublicKey:  signedPreKey.PublicKey[1:], // 移除 05
		PrivateKey: signedPreKey.PrivateKey,
		Signature:  signedPreKey.Signature,
		Timestamp:  proto.Uint64(uint64(signedPreKey.Timestamp)),
	}

	data, err := proto.Marshal(signedPreKeyRecordProto)
	if err != nil {
		return nil
	}
	return data
}

// Deserialize will take in JSON bytes and return a signed prekey record structure.
func (p *ProtoSignedPreKeyRecordSerializer) Deserialize(serialized []byte) (*record.SignedPreKeyStructure, error) {
	signedPreKeyRecord := &signalProto.SignedPreKeyRecordStructure{}
	err := proto.Unmarshal(serialized, signedPreKeyRecord)
	if err != nil {
		return nil, err
	}

	return &record.SignedPreKeyStructure{
		ID:         signedPreKeyRecord.GetId(),
		PublicKey:  signedPreKeyRecord.GetPublicKey(),
		PrivateKey: signedPreKeyRecord.GetPrivateKey(),
		Signature:  signedPreKeyRecord.GetSignature(),
		Timestamp:  int64(signedPreKeyRecord.GetTimestamp()),
	}, nil
}

// JSONStateSerializer is a structure for serializing session states into
// and from JSON.
type ProtoStateSerializer struct{}

// Serialize will take a session state structure and convert it to JSON bytes.
func (j *ProtoStateSerializer) Serialize(state *record.StateStructure) []byte {
	structure := makeSessionStructureToProto(state)
	d, err := proto.Marshal(structure)
	if err != nil {
		return []byte{}
	}
	return d
}

// Deserialize will take in JSON bytes and return a session state structure.
func (j *ProtoStateSerializer) Deserialize(serialized []byte) (*record.StateStructure, error) {
	sessionStruct := &signalProto.SessionStructure{}
	err := proto.Unmarshal(serialized, sessionStruct)
	if err != nil {
		return nil, err
	}

	return makeProtoToSessionStruct(sessionStruct), nil
}

type ProtoPreKeyRecordSerializer struct{}

// Serialize will take a prekey record structure and convert it to JSON bytes.
func (p *ProtoPreKeyRecordSerializer) Serialize(preKey *record.PreKeyStructure) []byte {
	protoPreKeyStructure := signalProto.PreKeyRecordStructure{
		PrivateKey: preKey.PrivateKey,
		PublicKey:  preKey.PublicKey[1:],
		Id:         proto.Uint32(preKey.ID),
	}
	data, err := proto.Marshal(&protoPreKeyStructure)
	if err != nil {
		return nil
	}
	return data
}

// Deserialize will take in JSON bytes and return a prekey record structure.
func (p *ProtoPreKeyRecordSerializer) Deserialize(serialized []byte) (*record.PreKeyStructure, error) {
	protoPreKeysStruct := signalProto.PreKeyRecordStructure{}
	err := proto.Unmarshal(serialized, &protoPreKeysStruct)
	if err != nil {
		return nil, err
	}

	return &record.PreKeyStructure{
		ID:         protoPreKeysStruct.GetId(),
		PublicKey:  protoPreKeysStruct.GetPublicKey(),
		PrivateKey: protoPreKeysStruct.GetPrivateKey(),
	}, nil
}

// JSONSessionSerializer is a structure for serializing session records into
// and from JSON.
type ProtoSessionSerializer struct{}

// Serialize will take a session structure and convert it to Proto bytes.
func (j *ProtoSessionSerializer) Serialize(session *record.SessionStructure) []byte {
	if session == nil {
		return []byte{}
	}
	recordStructure := &signalProto.RecordStructure{}
	if session.SessionState != nil {
		structure := makeSessionStructureToProto(session.SessionState)
		recordStructure.CurrentSession = structure
	}

	if session.PreviousStates != nil && len(session.PreviousStates) > 0 {
		recordStructure.PreviousSessions = make([]*signalProto.SessionStructure, 0)
		for _, state := range session.PreviousStates {
			recordStructure.PreviousSessions = append(recordStructure.PreviousSessions, makeSessionStructureToProto(state))
		}
	}

	d, _ := json.Marshal(session)
	//log.Println(string(d))

	d, err := proto.Marshal(recordStructure)
	if err != nil {
		return []byte{}
	}
	return d
}

// Deserialize will take in Proto bytes and return a session structure, which can be
// used to create a new Session Record object.
func (j *ProtoSessionSerializer) Deserialize(serialized []byte) (*record.SessionStructure, error) {
	recordStructure := &signalProto.RecordStructure{}
	err := proto.Unmarshal(serialized, recordStructure)
	if err != nil {
		return nil, err
	}

	//json.Marshal(recordStructure)
	//log.Println(string(d))

	s := &record.SessionStructure{
		SessionState:   makeProtoToSessionStruct(recordStructure.CurrentSession),
		PreviousStates: make([]*record.StateStructure, 0),
	}

	for _, structure := range recordStructure.GetPreviousSessions() {
		if structure.SessionVersion != nil {
			s.PreviousStates = append(s.PreviousStates, makeProtoToSessionStruct(structure))
		}
	}

	dd, _ := json.Marshal(s)
	log.Println("d->", string(dd))

	return s, nil
}

func makeProtoToSessionStructChain(protoChainStructure *signalProto.SessionStructure_Chain) *record.ChainStructure {
	chainStruct := &record.ChainStructure{}
	chainStruct.SenderRatchetKeyPublic = protoChainStructure.GetSenderRatchetKey()
	chainStruct.SenderRatchetKeyPrivate = protoChainStructure.GetSenderRatchetKeyPrivate()
	if protoChainStructure.GetChainKey() != nil {
		protoChainKey := protoChainStructure.GetChainKey()
		chainKey := &chain.KeyStructure{}
		chainKey.Index = protoChainKey.GetIndex()
		chainKey.Key = protoChainKey.GetKey()
		chainStruct.ChainKey = chainKey
	}

	if protoChainStructure.GetMessageKeys() != nil {
		messageKeys := make([]*message.KeysStructure, 0)
		for _, key := range protoChainStructure.GetMessageKeys() {
			k := &message.KeysStructure{}
			k.Index = key.GetIndex()
			k.IV = key.GetIv()
			k.MacKey = key.GetMacKey()
			k.CipherKey = key.GetCipherKey()
			messageKeys = append(messageKeys, k)
		}
		chainStruct.MessageKeys = messageKeys
	}
	return chainStruct
}

func makeProtoToSessionStruct(protoStructure *signalProto.SessionStructure) *record.StateStructure {
	stateStructure := &record.StateStructure{}
	if protoStructure != nil {
		stateStructure.SessionVersion = int(protoStructure.GetSessionVersion())
		stateStructure.NeedsRefresh = protoStructure.GetNeedsRefresh()
		stateStructure.LocalRegistrationID = protoStructure.GetLocalRegistrationId()
		stateStructure.RemoteRegistrationID = protoStructure.GetRemoteRegistrationId()
		stateStructure.PreviousCounter = protoStructure.GetPreviousCounter()
		stateStructure.RootKey = protoStructure.GetRootKey()
		stateStructure.RemoteIdentityPublic = protoStructure.GetRemoteIdentityPublic()
		stateStructure.LocalIdentityPublic = protoStructure.GetLocalIdentityPublic()
		stateStructure.SenderBaseKey = protoStructure.GetAliceBaseKey()
		if protoStructure.GetSenderChain() != nil {
			stateStructure.SenderChain = makeProtoToSessionStructChain(protoStructure.GetSenderChain())
		}

		if protoStructure.GetReceiverChains() != nil {
			chains := make([]*record.ChainStructure, 0)
			for _, structure_chain := range protoStructure.GetReceiverChains() {
				chains = append(chains, makeProtoToSessionStructChain(structure_chain))
			}
			stateStructure.ReceiverChains = chains
		}

		if protoStructure.GetPendingKeyExchange() != nil {
			protoPendingPreKey := protoStructure.GetPendingKeyExchange()
			p := &record.PendingKeyExchangeStructure{}
			p.Sequence = protoPendingPreKey.GetSequence()

			p.LocalRatchetKeyPublic = protoPendingPreKey.GetLocalRatchetKey()
			p.LocalRatchetKeyPrivate = protoPendingPreKey.GetLocalRatchetKeyPrivate()

			p.LocalBaseKeyPublic = protoPendingPreKey.GetLocalBaseKey()
			p.LocalBaseKeyPrivate = protoPendingPreKey.GetLocalBaseKeyPrivate()

			p.LocalIdentityKeyPublic = protoPendingPreKey.GetLocalIdentityKey()
			p.LocalIdentityKeyPrivate = protoPendingPreKey.GetLocalIdentityKeyPrivate()

			stateStructure.PendingKeyExchange = p
		}

		if protoStructure.GetPendingPreKey() != nil {
			protoPendingPreKey := protoStructure.GetPendingPreKey()
			pendingPrekey := record.PendingPreKeyStructure{}
			pendingPrekey.BaseKey = protoPendingPreKey.GetBaseKey()
			pendingPrekey.SignedPreKeyID = uint32(protoPendingPreKey.GetSignedPreKeyId())
			pendingPrekey.PreKeyID = optional.NewOptionalUint32(protoPendingPreKey.GetPreKeyId())

			stateStructure.PendingPreKey = &pendingPrekey
		}
	}
	return stateStructure
}

func makeSessionStructureToProto(state *record.StateStructure) *signalProto.SessionStructure {
	sessionStructure := &signalProto.SessionStructure{}
	if state != nil {
		sessionStructure.SessionVersion = proto.Uint32(uint32(state.SessionVersion))
		sessionStructure.LocalIdentityPublic = state.LocalIdentityPublic
		sessionStructure.RemoteIdentityPublic = state.RemoteIdentityPublic
		sessionStructure.RootKey = state.RootKey
		sessionStructure.PreviousCounter = proto.Uint32(state.PreviousCounter)
		if state.SenderChain != nil {
			sessionStructure.SenderChain = makeSessionStructureChainToProto(state.SenderChain)
		}

		if state.ReceiverChains != nil && len(state.ReceiverChains) > 0 {
			sessionStructureChains := make([]*signalProto.SessionStructure_Chain, 0)
			for _, receiverChain := range state.ReceiverChains {
				sessionStructureChains = append(sessionStructureChains, makeSessionStructureChainToProto(receiverChain))
			}
			sessionStructure.ReceiverChains = sessionStructureChains
		}

		if state.PendingKeyExchange != nil {
			keyExchange := state.PendingKeyExchange
			KeyExchangeProto := &signalProto.SessionStructure_PendingKeyExchange{}
			KeyExchangeProto.Sequence = proto.Uint32(keyExchange.Sequence)

			KeyExchangeProto.LocalBaseKey = keyExchange.LocalBaseKeyPublic
			KeyExchangeProto.LocalBaseKeyPrivate = keyExchange.LocalBaseKeyPrivate

			KeyExchangeProto.LocalIdentityKey = keyExchange.LocalIdentityKeyPublic
			KeyExchangeProto.LocalIdentityKeyPrivate = keyExchange.LocalIdentityKeyPrivate

			KeyExchangeProto.LocalRatchetKey = keyExchange.LocalRatchetKeyPublic
			KeyExchangeProto.LocalBaseKeyPrivate = keyExchange.LocalRatchetKeyPrivate

			sessionStructure.PendingKeyExchange = KeyExchangeProto
		}

		if state.PendingPreKey != nil {
			preKey := state.PendingPreKey
			preKeyProto := &signalProto.SessionStructure_PendingPreKey{}
			preKeyProto.BaseKey = preKey.BaseKey
			preKeyProto.PreKeyId = proto.Uint32(preKey.PreKeyID.Value)
			preKeyProto.SignedPreKeyId = proto.Int32(int32(preKey.SignedPreKeyID))

			sessionStructure.PendingPreKey = preKeyProto
		}
		sessionStructure.RemoteRegistrationId = proto.Uint32(state.RemoteRegistrationID)
		sessionStructure.LocalRegistrationId = proto.Uint32(state.LocalRegistrationID)
		sessionStructure.AliceBaseKey = state.SenderBaseKey
		sessionStructure.NeedsRefresh = proto.Bool(state.NeedsRefresh)
	}
	return sessionStructure
}

func makeSessionStructureChainToProto(structure *record.ChainStructure) *signalProto.SessionStructure_Chain {
	sChain := &signalProto.SessionStructure_Chain{}
	sChain.SenderRatchetKey = structure.SenderRatchetKeyPublic
	sChain.SenderRatchetKeyPrivate = structure.SenderRatchetKeyPrivate
	if structure.ChainKey != nil {
		chainKey := structure.ChainKey
		sChainKey := &signalProto.SessionStructure_Chain_ChainKey{}
		sChainKey.Key = chainKey.Key
		sChainKey.Index = proto.Uint32(chainKey.Index)
		// set chainkey
		sChain.ChainKey = sChainKey
	}

	if structure.MessageKeys != nil && len(structure.MessageKeys) > 0 {
		messageKeys := make([]*signalProto.SessionStructure_Chain_MessageKey, 0)
		for _, key := range structure.MessageKeys {
			messageKey := &signalProto.SessionStructure_Chain_MessageKey{}
			messageKey.Index = proto.Uint32(key.Index)
			messageKey.CipherKey = key.CipherKey
			messageKey.MacKey = key.MacKey
			messageKey.Iv = key.IV
			messageKeys = append(messageKeys, messageKey)
		}
		sChain.MessageKeys = messageKeys
	}

	return sChain
}

func makeSenderKeyStateToProto(state *groupRecord.SenderKeyStateStructure) *signalProto.SenderKeyStateStructure {
	protoSenderKeyState := &signalProto.SenderKeyStateStructure{}
	protoSenderKeyState.SenderKeyId = &state.KeyID
	protoSenderKeyState.SenderSigningKey = &signalProto.SenderKeyStateStructure_SenderSigningKey{
		Public:  state.SigningKeyPublic,
		Private: state.SigningKeyPrivate,
	}
	log.Println("makeSenderKeyStateToProto SigningKeyPrivate ", hex.EncodeToString(state.SigningKeyPrivate))
	if state.SenderChainKey != nil {
		protoSenderKeyState.SenderChainKey = &signalProto.SenderKeyStateStructure_SenderChainKey{
			Iteration: &state.SenderChainKey.Iteration,
			Seed:      state.SenderChainKey.ChainKey,
		}
	}

	if state.Keys != nil && len(state.Keys) > 0 {
		protoSenderKeyState.SenderMessageKeys = make([]*signalProto.SenderKeyStateStructure_SenderMessageKey, len(state.Keys))
		for i, key := range state.Keys {
			protoSenderKeyState.SenderMessageKeys[i] = &signalProto.SenderKeyStateStructure_SenderMessageKey{
				Iteration: &key.Iteration,
				Seed:      key.Seed,
			}
		}
	}
	return protoSenderKeyState
}

func makeSenderKeyToStruct(protoState *signalProto.SenderKeyStateStructure) *groupRecord.SenderKeyStateStructure {
	s := &groupRecord.SenderKeyStateStructure{}
	protoMessageKeys := protoState.GetSenderMessageKeys()
	if protoMessageKeys != nil && len(protoMessageKeys) != 0 {
		s.Keys = make([]*ratchet.SenderMessageKeyStructure, len(protoMessageKeys))
		for i, key := range protoMessageKeys {
			s.Keys[i] = &ratchet.SenderMessageKeyStructure{
				Iteration: key.GetIteration(),
				Seed:      key.GetSeed(),
			}
		}
	}
	protoSenderChainKey := protoState.GetSenderChainKey()
	if protoSenderChainKey != nil {
		s.SenderChainKey = &ratchet.SenderChainKeyStructure{
			Iteration: protoSenderChainKey.GetIteration(),
			ChainKey:  protoSenderChainKey.GetSeed(),
		}
	}
	s.KeyID = protoState.GetSenderKeyId()
	s.SigningKeyPrivate = protoState.GetSenderSigningKey().GetPrivate()
	log.Println("makeSenderKeyToStruct SigningKeyPrivate ", hex.EncodeToString(protoState.GetSenderSigningKey().GetPrivate()))
	s.SigningKeyPublic = protoState.GetSenderSigningKey().GetPublic()
	return s
}
