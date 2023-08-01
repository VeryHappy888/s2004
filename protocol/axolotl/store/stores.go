package store

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gtime"
	"github.com/gogf/gf/util/gconv"
	"log"
	"strings"
	"sync"
	"ws-go/libsignal/ecc"
	groupRecord "ws-go/libsignal/groups/state/record"
	"ws-go/libsignal/keys/identity"
	"ws-go/libsignal/protocol"
	"ws-go/libsignal/serialize"
	"ws-go/libsignal/state/record"
	"ws-go/libsignal/util/bytehelper"
	"ws-go/libsignal/util/keyhelper"
	"ws-go/protocol/axolotl/serializer"
)

// Define some in-memory stores for testing.

// IdentityKeyStore
func NewInMemoryIdentityKey(db gdb.DB, identityKey *identity.KeyPair, localRegistrationID uint32) *InMemoryIdentityKey {
	i := &InMemoryIdentityKey{
		identityStore:       db,
		trustedKeys:         make(map[*protocol.SignalAddress]*identity.Key),
		identityKeyPair:     identityKey,
		localRegistrationID: localRegistrationID,
		Lock:                sync.RWMutex{},
	}
	return i
}

type InMemoryIdentityKey struct {
	identityStore       gdb.DB
	trustedKeys         map[*protocol.SignalAddress]*identity.Key
	identityKeyPair     *identity.KeyPair
	localRegistrationID uint32
	Lock                sync.RWMutex
}

func (i *InMemoryIdentityKey) GetIdentityKeyPair() *identity.KeyPair {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetIdentityKeyPair", err)
		}
	}()
	if i.identityKeyPair != nil {
		return i.identityKeyPair
	}
	_ = i.queryMyIdentityKeys()
	return i.identityKeyPair
}

func (i *InMemoryIdentityKey) GetLocalRegistrationId() uint32 {
	// 先获取变量值如果有直接返回不查询数据
	if i.localRegistrationID != 0 {
		return i.localRegistrationID
	}

	_ = i.queryMyIdentityKeys()
	return i.localRegistrationID
}

func (i *InMemoryIdentityKey) SaveIdentity(address *protocol.SignalAddress, identityKey *identity.Key) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:SaveIdentity", err)
		}
	}()
	i.trustedKeys[address] = identityKey
	if i.identityStore != nil {
		identitiesModel := i.identityStore.Model("identities")
		// set inset Data
		insetData := gdb.Map{}
		insetData["recipient_id"] = address.Name()
		insetData["device_id"] = 0
		if identityKey != nil && identityKey.PublicKey() != nil {
			insetData["public_key"] = identityKey.PublicKey().Serialize()
		}
		count, err := identitiesModel.Where("recipient_id =?", address.Name()).Count()
		if err != nil {
			log.Println("countIdentity error", err)
			return
		}
		if count > 0 {
			_, err := identitiesModel.Where("recipient_id =?", address.Name()).Delete()
			if err != nil {
				log.Println("DeleteIdentity error", err)
				return
			}
		}
		// insert
		_, err = identitiesModel.Insert(insetData)
		if err != nil {
			log.Println("SaveIdentity error", err)
			return
		}
	}
}

func (i *InMemoryIdentityKey) IsTrustedIdentity(address *protocol.SignalAddress, identityKey *identity.Key) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:IsTrustedIdentity", err)
		}
	}()
	for signalAddress, trusted := range i.trustedKeys {
		if strings.Contains(signalAddress.Name(), address.Name()) {
			return trusted == nil || trusted.Fingerprint() == identityKey.Fingerprint()
		}
	}

	// query data base
	if i.identityStore != nil {
		identitiesModel := i.identityStore.Model("identities")
		oneRecord, err := identitiesModel.Where("recipient_id=?", address.Name()).FindOne()
		if err != nil {
			log.Println("IsTrustedIdentity err", err)
			return true
		}
		// public keys
		if v, ok := oneRecord["public_key"]; ok {
			dePubKey, _ := ecc.DecodePoint(v.Bytes(), 0)
			djbECPublicKey := ecc.NewDjbECPublicKey(dePubKey.PublicKey())
			trusted := identity.NewKey(djbECPublicKey)
			return trusted == nil || trusted.Fingerprint() == identityKey.Fingerprint()
		}
	}

	return true
}

// queryMyIdentityKeys
func (i *InMemoryIdentityKey) queryMyIdentityKeys() error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:queryMyIdentityKeys", err)
		}
	}()
	// query data base
	if i.identityStore != nil {
		identitiesModel := i.identityStore.Model("identities")
		oneRecord, err := identitiesModel.Where("recipient_id=?", -1).FindOne()
		if err != nil {
			log.Println("GetLocalRegistrationId err", err)
			return err
		}
		// registration id
		if v, ok := oneRecord["registration_id"]; ok {
			i.localRegistrationID = v.Uint32()
		}

		var (
			djbECPublicKey  *ecc.DjbECPublicKey
			djbECPrivateKey *ecc.DjbECPrivateKey
		)
		// public keys
		if v, ok := oneRecord["public_key"]; ok {
			dePubKey, _ := ecc.DecodePoint(v.Bytes(), 0)
			djbECPublicKey = ecc.NewDjbECPublicKey(dePubKey.PublicKey())
		}
		// pri keys
		if v, ok := oneRecord["private_key"]; ok {
			djbECPrivateKey = ecc.NewDjbECPrivateKey(bytehelper.SliceToArray(v.Bytes()))
		}
		// set identity keys
		i.identityKeyPair = identity.NewKeyPair(identity.NewKey(djbECPublicKey), djbECPrivateKey)
	}
	return nil
}

// saveMyIdentityKeys
func (i *InMemoryIdentityKey) saveMyIdentityKeys(pair *identity.KeyPair, regId uint32) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:saveMyIdentityKeys", err)
		}
	}()
	if i.identityStore != nil {
		identitiesModel := i.identityStore.Model("identities")
		// set inset Data
		insetData := gdb.Map{}
		insetData["recipient_id"] = -1
		insetData["device_id"] = 0
		// registration id
		insetData["registration_id"] = regId
		// update time
		insetData["timestamp"] = gtime.TimestampMilli()
		// public key
		if pair != nil && pair.PublicKey() != nil {
			insetData["public_key"] = pair.PublicKey().PublicKey().Serialize()
		}
		// pri key
		if pair != nil && pair.PrivateKey() != nil {
			insetData["private_key"] = bytehelper.ArrayToSlice(pair.PrivateKey().Serialize())
		}
		// insert
		_, err := identitiesModel.Insert(insetData)
		if err != nil {
			return err
		}

		// set vars
		i.localRegistrationID = regId
		i.identityKeyPair = pair
	}

	return nil
}

// PreKeyStore
func NewInMemoryPreKey(db gdb.DB) *InMemoryPreKey {
	return &InMemoryPreKey{
		storesDB: db,
		store:    make(map[uint32]*record.PreKey),
		Lock:     sync.RWMutex{},
	}
}

type InMemoryPreKey struct {
	storesDB gdb.DB
	store    map[uint32]*record.PreKey
	Lock     sync.RWMutex
}

func (i *InMemoryPreKey) LoadPreKey(preKeyID uint32) *record.PreKey {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:LoadPreKey", err)
		}
	}()
	if i.storesDB != nil {
		preKeysModel := i.storesDB.Model("prekeys")
		result, err := preKeysModel.FindOne("prekey_id=?", preKeyID)
		if err != nil {
			log.Println("InMemoryPreKey LoadPreKey error", err)
			return i.store[preKeyID]
		}
		// is  empty
		if result.IsEmpty() {
			return i.store[preKeyID]
		}
		// record
		if recordValue, ok := result["record"]; ok {
			preKey, err := record.NewPreKeyFromBytes(recordValue.Bytes(), &serializer.ProtoPreKeyRecordSerializer{})
			if err != nil {
				return i.store[preKeyID]
			}
			return preKey
		}
	}
	return i.store[preKeyID]
}

func (i *InMemoryPreKey) StorePreKeyIds(preKeys []*record.PreKey) {
	defer func() {
		if err := recover(); err != nil {
			i.Lock.Unlock()
			fmt.Println("run web error:StorePreKey", err)
		}
	}()
	list := g.Slice{}
	for _, key := range preKeys {
		i.Lock.Lock()
		i.store[key.ID().Value] = key
		if i.storesDB != nil {
			anyMap := g.Map{
				"prekey_id":           key.ID().Value,
				"record":              key.Serialize(),
				"sent_to_server":      0,
				"direct_distribution": false,
				"upload_timestamp":    gtime.TimestampMilli(),
			}
			list = append(list, anyMap)
		}
		i.Lock.Unlock()
	}
	if i.storesDB != nil && len(list) > 0 {
		preKeysModel := i.storesDB.Model("prekeys")
		_, err := preKeysModel.Insert(list)
		if err != nil {
			log.Println("go InMemoryPreKey StorePreKey error", err)
			return
		}
	}
}

func (i *InMemoryPreKey) StorePreKey(preKeyID uint32, preKeyRecord *record.PreKey) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:StorePreKey", err)
		}
	}()
	i.Lock.Lock()
	defer i.Lock.Unlock()
	i.store[preKeyID] = preKeyRecord
	if i.storesDB != nil {
		preKeysModel := i.storesDB.Model("prekeys")
		anyMap := gdb.Map{ //并发
			"prekey_id":           preKeyID,
			"record":              preKeyRecord.Serialize(),
			"sent_to_server":      0,
			"direct_distribution": false,
			"upload_timestamp":    gtime.TimestampMilli(),
		}
		_, err := preKeysModel.Insert(anyMap) //并发读
		if err != nil {
			log.Println("InMemoryPreKey StorePreKey error", err)
			return
		}
	}
}

func (i *InMemoryPreKey) ContainsPreKey(preKeyID uint32) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:ContainsPreKey", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	_, ok := i.store[preKeyID]
	if i.storesDB != nil {
		preKeysModel := i.storesDB.Model("prekeys")
		isExist, err := preKeysModel.Count("prekey_id=?", preKeyID)
		if err != nil {
			log.Println("InMemoryPreKey ContainsPreKey error", err)
			return false
		}
		if isExist > 0 {
			return true
		}
	}
	return ok
}

func (i *InMemoryPreKey) RemovePreKey(preKeyID uint32) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:RemovePreKey", err)
		}
	}()
	delete(i.store, preKeyID)
	if i.storesDB != nil {
		preKeysModel := i.storesDB.Model("prekeys")
		_, err := preKeysModel.Delete("prekey_id=?", preKeyID)
		if err != nil {
			log.Println("InMemoryPreKey RemovePreKey error", err)
			return
		}
	}
}

// SessionStore
func NewInMemorySession(serializer *serialize.Serializer, db gdb.DB) *InMemorySession {
	return &InMemorySession{
		storesDB:   db,
		sessions:   make(map[*protocol.SignalAddress]*record.Session),
		serializer: serializer,
		Lock:       sync.RWMutex{},
	}
}

type InMemorySession struct {
	storesDB   gdb.DB
	sessions   map[*protocol.SignalAddress]*record.Session
	serializer *serialize.Serializer
	Lock       sync.RWMutex
}

func (i *InMemorySession) LoadSession(address *protocol.SignalAddress) *record.Session {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:LoadSession", err)
		}
	}()
	i.Lock.Lock()
	defer i.Lock.Unlock()
	for signalAddress, session := range i.sessions {
		if strings.Contains(signalAddress.Name(), address.Name()) {
			return session
		}
	}

	if i.storesDB != nil {
		sessionsModel := i.storesDB.Model("sessions")
		sessionsModel.Where("recipient_id=?", gconv.Int64(address.Name()))
		session, err := sessionsModel.FindOne()
		if err != nil {
			session := record.NewSession(i.serializer.Session, i.serializer.State)
			i.sessions[address] = session
			return session
		}
		// is empty
		if session.IsEmpty() {
			session := record.NewSession(i.serializer.Session, i.serializer.State)
			i.sessions[address] = session
			return session
		}
		// get record
		if v, ok := session["record"]; !ok {
			session := record.NewSession(i.serializer.Session, i.serializer.State)
			i.sessions[address] = session
			return session
		} else {
			protoSessionSerializer := serializer.ProtoSessionSerializer{}
			protoStateSerializer := serializer.ProtoStateSerializer{}
			log.Println("LoadSession:", hex.EncodeToString(v.Bytes()))
			session, err := record.NewSessionFromBytes(v.Bytes(), &protoSessionSerializer, &protoStateSerializer)
			if err != nil {
				return nil
			}
			return session
		}
	}

	session := record.NewSession(i.serializer.Session, i.serializer.State)
	//i.sessions[address] = session
	return session
}

func (i *InMemorySession) GetSubDeviceSessions(name string) []uint32 {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetSubDeviceSessions", err)
		}
	}()
	var deviceIDs []uint32

	for key := range i.sessions {
		if key.Name() == name && key.DeviceID() != 1 {
			deviceIDs = append(deviceIDs, key.DeviceID())
		}
	}

	return deviceIDs
}

func (i *InMemorySession) StoreSession(remoteAddress *protocol.SignalAddress, record *record.Session) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:StoreSession", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	i.sessions[remoteAddress] = record
	if i.storesDB != nil {
		sessionsModel := i.storesDB.Model("sessions")
		sessionsModel.Where("recipient_id=?", gconv.Int64(remoteAddress.Name()))
		isExist, err := sessionsModel.Count()
		if err != nil {
			return
		}
		// recipient_id
		name := remoteAddress.Name()
		// record bytes
		recordData := record.Serialize()
		log.Println("StoreSession recordData:", hex.EncodeToString(recordData))
		// if exist only update if not exist inset
		if isExist > 0 {
			anyMap := gdb.Map{
				"record": recordData,
			}
			_, err = sessionsModel.Data(anyMap).
				Where("recipient_id", gconv.Int64(name)).
				Update()

			if err != nil {
				log.Println("StoreSession update error recipient_id = ", name, err)
			}
		} else {
			anyMap := gdb.Map{
				"recipient_id": gconv.Int64(name),
				"record":       recordData,
				"device_id":    0,
				"timestamp":    gtime.Timestamp(),
			}
			_, err = sessionsModel.Insert(anyMap)
			/**/
			if err != nil {
				log.Println("StoreSession Save error recipient_id = ", name, err)
			}
		}

	}
	log.Println("StoreSession", hex.EncodeToString(record.Serialize()))
}

func (i *InMemorySession) ContainsSession(remoteAddress *protocol.SignalAddress) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:ContainsSession", err)
		}
	}()
	for signalAddress, _ := range i.sessions {
		if strings.Contains(signalAddress.Name(), remoteAddress.Name()) {
			return true
		}
	}

	if i.storesDB != nil {
		i.Lock.RLock()
		sessionsModel := i.storesDB.Model("sessions")
		sessionsModel.Where("recipient_id", gconv.Int64(remoteAddress.Name()))
		isExist, err := sessionsModel.Count()
		i.Lock.RUnlock()
		if err != nil {
			log.Println("ContainsSession where error", err)
			return false
		}
		// is exist
		if isExist > 0 {
			return true
		}
	}
	return false
}

func (i *InMemorySession) DeleteSession(remoteAddress *protocol.SignalAddress) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:DeleteSession", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	delete(i.sessions, remoteAddress)
}

func (i *InMemorySession) DeleteAllSessions() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:DeleteAllSessions", err)
		}
	}()
	i.sessions = make(map[*protocol.SignalAddress]*record.Session)
}

// SignedPreKeyStore
func NewInMemorySignedPreKey(db gdb.DB) *InMemorySignedPreKey {
	return &InMemorySignedPreKey{
		signedDatabase: db,
		store:          make(map[uint32]*record.SignedPreKey),
		Lock:           sync.RWMutex{},
	}
}

type InMemorySignedPreKey struct {
	signedDatabase gdb.DB
	store          map[uint32]*record.SignedPreKey
	Lock           sync.RWMutex
}

func (i *InMemorySignedPreKey) LoadSignedPreKey(signedPreKeyID uint32) *record.SignedPreKey {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:LoadSignedPreKey", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	//if sPreKeys,ok := i.store[signedPreKeyID];ok {
	//	return sPreKeys
	//}
	// query data base
	if i.signedDatabase != nil {
		signedPreKeysModel := i.signedDatabase.Model("signed_prekeys")
		oneRecord, err := signedPreKeysModel.FindOne("prekey_id=?", signedPreKeyID)
		if err != nil {
			log.Println("LoadSignedPreKey error", err)
			return nil
		}
		// record
		if recordValue, ok := oneRecord["record"]; ok {
			sPreKeys, err := record.NewSignedPreKeyFromBytes(recordValue.Bytes(), &serializer.ProtoSignedPreKeyRecordSerializer{})
			if err != nil {
				log.Println("LoadSignedPreKey error", err)
				return nil
			}
			return sPreKeys
		}
	}
	return i.store[signedPreKeyID]
}

func (i *InMemorySignedPreKey) LoadSignedPreKeys() []*record.SignedPreKey {

	var preKeys []*record.SignedPreKey

	for _, record := range i.store {
		preKeys = append(preKeys, record)
	}

	return preKeys
}

func (i *InMemorySignedPreKey) StoreSignedPreKey(signedPreKeyID uint32, record *record.SignedPreKey) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:StoreSignedPreKey", err)
		}
	}()
	//i.store[signedPreKeyID] = record
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	if i.signedDatabase != nil {
		signedPreKeysModel := i.signedDatabase.Model("signed_prekeys")
		saveData := gdb.Map{}
		saveData["prekey_id"] = signedPreKeyID
		saveData["timestamp"] = gtime.TimestampMilli()
		saveData["record"] = record.Serialize()
		count, err := signedPreKeysModel.Count()
		if err != nil {
			log.Println("StoreSignedPreKey count error", err)
			return
		}
		if count > 0 {
			_, err = signedPreKeysModel.Delete()
			if err != nil {
				log.Println("StoreSignedPreKey delete error", err)
				return
			}
		}
		_, err = signedPreKeysModel.Insert(saveData)
		if err != nil {
			log.Println("StoreSignedPreKey Insert error", err)
			return
		}
	}
}

func (i *InMemorySignedPreKey) ContainsSignedPreKey(signedPreKeyID uint32) bool {
	_, ok := i.store[signedPreKeyID]
	return ok
}

func (i *InMemorySignedPreKey) RemoveSignedPreKey(signedPreKeyID uint32) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:RemoveSignedPreKey", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	delete(i.store, signedPreKeyID)
	if i.signedDatabase != nil {
		signedPreKeysModel := i.signedDatabase.Model("signed_prekeys")
		_, err := signedPreKeysModel.Delete("prekey_id=?", signedPreKeyID)
		if err != nil {
			log.Println("RemoveSignedPreKey error ", err)
		}
	}
}

func NewInMemorySenderKey(db gdb.DB) *InMemorySenderKey {
	return &InMemorySenderKey{
		storesDB: db,
		store:    make(map[*protocol.SenderKeyName]*groupRecord.SenderKey),
		Lock:     sync.RWMutex{},
	}
}

type InMemorySenderKey struct {
	storesDB gdb.DB
	store    map[*protocol.SenderKeyName]*groupRecord.SenderKey
	Lock     sync.RWMutex
}

func (i *InMemorySenderKey) StoreSenderKey(senderKeyName *protocol.SenderKeyName, keyRecord *groupRecord.SenderKey) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:StoreSenderKey", err)
		}
	}()
	//i.storeDataBase[senderKeyName] = keyRecord
	if i.storesDB != nil {
		senderKeyModel := i.storesDB.Model("sender_keys")
		senderId := gconv.Int64(senderKeyName.Sender().Name())
		senderKeyModel.Where("group_id=? AND sender_id=?", senderKeyName.GroupID(), senderId)
		i.Lock.RLock()
		isExist, err := senderKeyModel.Count()
		i.Lock.RUnlock()
		if err != nil {
			log.Println("StoreSenderKey databases error", err)
			return
		}
		recordData := keyRecord.Serialize()
		// is Exist
		if isExist > 0 {
			// update
			anyMap := gdb.Map{"record": recordData}
			_, err = senderKeyModel.Data(anyMap).
				Where("group_id=? AND sender_id=?", senderKeyName.GroupID(), senderId).
				Update()

			if err != nil {
				log.Println("StoreSenderKey update error group_id = ", senderKeyName.GroupID(), err)
			}
		} else {
			anyMap := gdb.Map{
				"group_id":  senderKeyName.GroupID(),
				"sender_id": senderId,
				"record":    recordData,
			}
			_, err = senderKeyModel.Insert(anyMap)
			if err != nil {
				log.Println("StoreSenderKey Save error group_id = ", senderKeyName.GroupID(), err)
			}
		}
	}
}

func (i *InMemorySenderKey) LoadSenderKey(senderKeyName *protocol.SenderKeyName) *groupRecord.SenderKey {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:LoadSenderKey", err)
		}
	}()
	i.Lock.RLock()
	defer i.Lock.RUnlock()
	// 先在内存中查找
	//for v, key := range i.storeDataBase {
	//	if strings.Contains(v.GroupID(), senderKeyName.GroupID()) &&
	//		strings.Contains(v.Sender().Name(), senderKeyName.Sender().Name()) {
	//		return key
	//	}
	//}
	// id
	groupId := senderKeyName.GroupID()
	senderId := gconv.Int64(senderKeyName.Sender().Name())
	// stores database
	if i.storesDB != nil {
		senderKeysModel := i.storesDB.Model("sender_keys")
		senderKeysModel.Where("group_id=? AND sender_id=?", groupId, senderId)
		oneData, err := senderKeysModel.FindOne()
		if err != nil {
			log.Println("LoadSenderKey findOne error", err)
			return nil
		}
		// is Empty
		if oneData.IsEmpty() {
			return nil
		}
		// recordValue
		if recordValue, ok := oneData["record"]; ok {
			senderKeySessionSerializer := &serializer.ProtoSenderKeySessionSerializer{}
			sendKeyStateSerializer := &serializer.ProtoSenderKeyStateSerializer{}
			senderKey, err := groupRecord.NewSenderKeyFromBytes(recordValue.Bytes(), senderKeySessionSerializer, sendKeyStateSerializer)
			if err != nil {
				return nil
			}
			return senderKey
		}
	}
	return nil
}

// SignalStore
type SignalStore struct {
	storeDataBase     gdb.DB
	SessionStore      *InMemorySession
	PreKeyStore       *InMemoryPreKey
	SignedPreKeyStore *InMemorySignedPreKey
	IdentityStore     *InMemoryIdentityKey
	SenderKeyStore    *InMemorySenderKey

	Serialize *serialize.Serializer
}

func (s *SignalStore) initSignalStore(staticPubKey string, staticPriKey string) error {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("initSignalStore panic: %v\n", r)
		}
	}()
	var (
		err             error
		signedPreKey    *record.SignedPreKey
		identityKeyPair *identity.KeyPair
		registrationID  uint32
	)
	// get serializer
	signedPreKeyRecordSerializer := &serializer.ProtoSignedPreKeyRecordSerializer{}
	// Generate an identity keypair
	identityKeyPair, err = keyhelper.GenerateIdentityKeyPair()
	if err != nil {
		return nil
	}
	if staticPriKey != "" && staticPubKey != "" {
		public, _ := base64.RawURLEncoding.DecodeString(staticPubKey)
		if len(public) == 0 || len(public) != 32 {
			public, _ = base64.RawURLEncoding.DecodeString(staticPubKey)
		}
		if len(public) == 0 || len(public) != 32 {
			public, _ = base64.StdEncoding.DecodeString(staticPubKey)
		}
		private, _ := base64.StdEncoding.DecodeString(staticPriKey)
		if len(private) == 0 || len(private) != 32 {
			private, _ = base64.RawStdEncoding.DecodeString(staticPriKey)
		}
		if len(private) == 0 || len(private) != 32 {
			private, _ = base64.RawURLEncoding.DecodeString(staticPriKey)
		}
		pu, _ := ecc.DecodePoint(public, 0)
		pr, _ := ecc.DecodePoint(private, 0)
		ecPrivateKey := ecc.NewECKeyPair(ecc.NewDjbECPublicKey(pu.PublicKey()), ecc.NewDjbECPrivateKey(pr.PublicKey()))
		identityKeyPair = identity.NewKeyPair(identity.NewKeyFromBytes(ecPrivateKey.PublicKey().PublicKey(), 0), ecPrivateKey.PrivateKey())
	}
	// Generate an  registration id
	registrationID = keyhelper.GenerateRegistrationID()
	//  Generate Signed PreKey
	signedPreKey, err = keyhelper.GenerateSignedPreKey(identityKeyPair, 0, signedPreKeyRecordSerializer)
	if err != nil {
		return nil
	}
	// set registration id and identityKeys to identity storeDataBase
	if err := s.IdentityStore.saveMyIdentityKeys(identityKeyPair, registrationID); err != nil {
		return err
	}
	// save signed pre keys
	s.SignedPreKeyStore.StoreSignedPreKey(
		signedPreKey.ID(),
		record.NewSignedPreKey(
			signedPreKey.ID(),
			signedPreKey.Timestamp(),
			signedPreKey.KeyPair(),
			signedPreKey.Signature(),
			signedPreKeyRecordSerializer,
		),
	)
	return nil
}

// NewSignalStore
func NewSignalStore(db gdb.DB, needInit bool, staticPubKey string, staticPriKey string) *SignalStore {
	// serialize
	newProtoSerializer := serializer.NewProtoSerializer()
	// create instance
	s := &SignalStore{
		storeDataBase:     db,
		SessionStore:      NewInMemorySession(newProtoSerializer, db),
		PreKeyStore:       NewInMemoryPreKey(db),
		SignedPreKeyStore: NewInMemorySignedPreKey(db),
		IdentityStore:     NewInMemoryIdentityKey(db, nil, 0),
		SenderKeyStore:    NewInMemorySenderKey(db),

		Serialize: newProtoSerializer,
	}
	// need init
	if needInit {
		_ = s.initSignalStore(staticPubKey, staticPriKey)
	}

	// Generate a registration id
	//
	//priKeyData, _ := hex.DecodeString("70d39c86f3e17e3ac93da9a53541dcdda5647bf618463499b48b0d61e5aa2c7a")
	//djbECPrivateKey := ecc.NewDjbECPrivateKey(bytehelper.SliceToArray(priKeyData))
	//
	//pubKeyData, _ := hex.DecodeString("053bbb934014e04dd100a874e5a328b536788d4fe7aba4f092b301d6584e2ac81e")
	//dePubKey, _ := ecc.DecodePoint(pubKeyData, 0)
	//djbECPublicKey := ecc.NewDjbECPublicKey(dePubKey.PublicKey())
	//log.Println(pubKeyData)
	//
	//newKeyPair := identity.NewKeyPair(identity.NewKey(djbECPublicKey), djbECPrivateKey)
	return s
}
