package axolotl

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/os/gtime"
	"github.com/gogf/gf/util/grand"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
	"ws-go/libsignal/ecc"
	"ws-go/libsignal/groups"
	"ws-go/libsignal/keys/prekey"
	"ws-go/libsignal/protocol"
	"ws-go/libsignal/session"
	"ws-go/libsignal/state/record"
	"ws-go/protocol/axolotl/serializer"
	"ws-go/protocol/axolotl/store"
	"ws-go/protocol/define"
)

const (
	maxKeys = 812

	sqlCreateIdentities      = "CREATE TABLE identities (_id INTEGER PRIMARY KEY AUTOINCREMENT, recipient_id INTEGER UNIQUE, device_id INTEGER, registration_id INTEGER, public_key BLOB, private_key BLOB, next_prekey_id INTEGER, timestamp INTEGER)"
	sqlCreateIdentitiesIndex = "CREATE UNIQUE INDEX IF NOT EXISTS identities_idx ON identities(recipient_id, device_id)"
	sqlCreatePreKeys         = "CREATE TABLE prekeys (_id INTEGER PRIMARY KEY AUTOINCREMENT, prekey_id INTEGER UNIQUE, sent_to_server BOOLEAN, record BLOB, direct_distribution BOOLEAN, upload_timestamp INTEGER)"
	sqlCreateSession         = "CREATE TABLE sessions (_id INTEGER PRIMARY KEY AUTOINCREMENT, recipient_id INTEGER, device_id INTEGER, record BLOB, timestamp INTEGER)"
	sqlCreateSenderKeys      = "CREATE TABLE sender_keys (_id INTEGER PRIMARY KEY AUTOINCREMENT, group_id TEXT NOT NULL, sender_id INTEGER NOT NULL, record BLOB NOT NULL)"
	sqlCreateSignedPreKeys   = "CREATE TABLE signed_prekeys (_id INTEGER PRIMARY KEY AUTOINCREMENT, prekey_id INTEGER UNIQUE, timestamp INTEGER, record BLOB)"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Manager struct {
	axolotlDB gdb.DB
	*store.SignalStore
	sessionCiphers map[string]*session.Cipher
	groupSession   map[string]*groups.GroupCipher
	groupBuilder   *groups.SessionBuilder
	Lock           sync.RWMutex
}

var createAxolotlTablesLock sync.Mutex

// createAxolotlTables 新建表
func createAxolotlTables(db gdb.DB) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:createAxolotlTables", err)
		}
	}()
	if db == nil {
		return
	}
	createAxolotlTablesLock.Lock()
	defer createAxolotlTablesLock.Unlock()
	// create tables
	// create identities table
	_, err := db.Exec(sqlCreateIdentities)
	if err != nil {
		log.Println("createAxolotlTables: create identities", err)
		return
	}
	// create identities index
	_, err = db.Exec(sqlCreateIdentitiesIndex)
	if err != nil {
		log.Println("createAxolotlTables: create identities index", err)
		return
	}
	// create preKeys tables
	_, err = db.Exec(sqlCreatePreKeys)
	if err != nil {
		log.Println("createAxolotlTables: create preKeys", err)
		return
	}
	// create session tables
	_, err = db.Exec(sqlCreateSession)
	if err != nil {
		log.Println("createAxolotlTables: create session", err)
		return
	}
	// create senderkeys table
	_, err = db.Exec(sqlCreateSenderKeys)
	if err != nil {
		log.Println("createAxolotlTables: create senderkeys", err)
		return
	}
	// create signed preKeys
	_, err = db.Exec(sqlCreateSignedPreKeys)
	if err != nil {
		log.Println("createAxolotlTables: create senderkeys", err)
		return
	}
}

// checkAxolotlDatabase
func checkAxolotlDatabase(u string) bool {
	dbPath := fmt.Sprintf("%s/%s/axolotl", define.DefaultDbPath, u)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}
	return true
}

//判断文件或文件夹是否存在
func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// initAxolotlDB
func initAxolotlDB(u string) (gdb.DB, bool, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:initAxolotlDB", err)
		}
	}()
	var axolotlDb gdb.DB
	dbPath := fmt.Sprintf("%s/%s/axolotl", define.DefaultDbPath, u)
	// init databases
	gdb.AddConfigNode(u, gdb.ConfigNode{
		Type:    "sqlite",
		Charset: "utf8",
		Link:    dbPath,
	})
	// connect databases
	if db, err := gdb.New(u); err != nil {
		log.Println(err.Error())
		return nil, false, err
	} else {
		axolotlDb = db
	}
	var needInit bool
	// exist databases file
	if !checkAxolotlDatabase(u) {
		if !isExist(define.DefaultDbPath + "/" + u) {
			err := os.Mkdir(define.DefaultDbPath+"/"+u, 0777)
			if err != nil {
				log.Println("mkdir error ", err.Error())
				return nil, false, err
			}
		}
		// create tables
		createAxolotlTables(axolotlDb)
		// needInit
		needInit = true
	}
	return axolotlDb, needInit, nil
}

// NewAxolotlManager
func NewAxolotlManager(u string, staticPubKey string, staticPriKey string) (*Manager, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:NewAxolotlManager", err)
		}
	}()
	m := &Manager{
		sessionCiphers: make(map[string]*session.Cipher, 0),
		groupSession:   make(map[string]*groups.GroupCipher, 0),
		Lock:           sync.RWMutex{},
	}
	// init databases
	db, needInit, err := initAxolotlDB(u)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// set databases
	m.axolotlDB = db
	m.SignalStore = store.NewSignalStore(db, needInit, staticPubKey, staticPriKey)
	m.groupBuilder = groups.NewGroupSessionBuilder(m.SenderKeyStore, m.Serialize)
	// gen keys
	if needInit && m.GetAllPreKeys() <= 0 {
		m.GeneratingPreKeys()
	}
	return m, nil
}

// GeneratingPreKeys
func (m *Manager) GeneratingPreKeys() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GeneratingPreKeys", err)
		}
	}()
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	keysCount := m.GetUnSentPreKeysCount(0)
	if keysCount >= maxKeys {
		log.Println("skipping key generation because already more than ", keysCount, "are unsent")
		return
	}
	var preKeys []*record.PreKey
	p := &serializer.ProtoPreKeyRecordSerializer{}

	last := grand.Intn(0xffffff) - 1
	maxCount := 812
	minCount := 0

	// gen keys
	for true {
		maxCount -= minCount
		if maxCount > 0 {
			if maxCount > 50 {
				minCount = 50
			} else {
				minCount = maxCount
			}
			for i := 0; i < minCount; i++ {
				key, err := ecc.GenerateKeyPair()
				if err != nil {
					return
				}
				preKeys = append(preKeys, record.NewPreKey(uint32(last+i+1), key, p))
				//log.Println(last + i + 1)
			}
			//log.Println("-------------------")
			last += minCount + 1
		} else {
			break
		}
	}

	//log.Println("1111")
	m.PreKeyStore.StorePreKeyIds(preKeys)
	// save
	/*for _, key := range preKeys {
		m.PreKeyStore.StorePreKey(key.ID().Value, key)
	}*/
}
func (m *Manager) UpdatePreKeysSent(ids []int) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:UpdatePreKeysSent", err)
		}
	}()
	if m.axolotlDB == nil {
		return errors.New("data bases not init")
	}
	m.Lock.Lock()
	defer m.Lock.Unlock()
	preKeysModel := m.axolotlDB.Model("prekeys")
	anyMap := gdb.Map{"sent_to_server": 1, "upload_timestamp": gtime.TimestampMilli()}
	_, err := preKeysModel.
		Where("prekey_id IN(?)", ids).
		Data(anyMap).
		Update()
	if err != nil {
		return err
	}
	return nil
}

func (m Manager) GetPreKeys() ([]*record.PreKey, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetPreKeys", err)
		}
	}()
	preKeys := make([]*record.PreKey, 0)
	if m.axolotlDB != nil {
		preKeysModel := m.axolotlDB.Model("prekeys")
		all, err := preKeysModel.FindAll()
		if err != nil {
			return nil, err
		}
		// all
		protoPreKeyRecordSerializer := &serializer.ProtoPreKeyRecordSerializer{}
		for _, r := range all {
			if recordValue, ok := r["record"]; ok {
				preKey, err := record.NewPreKeyFromBytes(recordValue.Bytes(), protoPreKeyRecordSerializer)
				if err != nil {
					return nil, err
				}
				preKeys = append(preKeys, preKey)
			}
		}
	}
	return preKeys, nil
}

// LoadUnSendPreKey
func (m *Manager) LoadUnSendPreKey() ([]*record.PreKey, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetPreKeys", err)
		}
	}()
	preKeys := make([]*record.PreKey, 0)
	if m.axolotlDB != nil {
		m.Lock.RLock()
		defer m.Lock.RUnlock()
		preKeysModel := m.axolotlDB.Model("prekeys")
		all, err := preKeysModel.FindAll("sent_to_server = 0")
		if err != nil {
			return nil, err
		}
		// all
		protoPreKeyRecordSerializer := &serializer.ProtoPreKeyRecordSerializer{}
		for _, r := range all {
			if recordValue, ok := r["record"]; ok {
				preKey, err := record.NewPreKeyFromBytes(recordValue.Bytes(), protoPreKeyRecordSerializer)
				if err != nil {
					return nil, err
				}
				preKeys = append(preKeys, preKey)
			}
		}
	}

	return preKeys, nil
}

// GetUnSentPreKeysCount
func (m *Manager) GetAllPreKeys() int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetPreKeys", err)
		}
	}()
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	if m.axolotlDB == nil {
		return -1
	}
	preKeysModel := m.axolotlDB.Model("prekeys")
	all, err := preKeysModel.Count()
	if err != nil {
		log.Println("AxolotlManager HasUnsentPreKeys error", err)
		return -1
	}
	return all
}

// GetUnSentPreKeysCount
func (m *Manager) GetUnSentPreKeysCount(sent int) int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("run web error:GetPreKeys", err)
		}
	}()
	if m.axolotlDB == nil {
		return -1
	}
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	preKeysModel := m.axolotlDB.Model("prekeys")
	unSentCount, err := preKeysModel.Where("sent_to_server = ? AND direct_distribution = 0", sent).Count()
	if err != nil {
		log.Println("AxolotlManager HasUnsentPreKeys error", err)
		return -1
	}
	return unSentCount
}

// HasUnSentPreKeys 有未发送的PreKeys
func (m *Manager) HasUnsentPreKeys() bool {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	unSentCount := m.GetUnSentPreKeysCount(0)
	return unSentCount != 0
}

// ContainsSession 是否创建了session 如果没有创建session 需要发送 下面的xxmp 来获取 prekeys
// 然后创建使用 createSession 创建session
//<iq id="02" xmlns="encrypt" type="get" to="@s.whatsapp.net">
//    <key>
//        <user jid="8618665179087@s.whatsapp.net"/>
//    </key>
//</iq>
func (m *Manager) ContainsSession(id string) bool {
	return m.SessionStore.ContainsSession(protocol.NewSignalAddress(id, 0))
}

// CreateSession
func (m *Manager) CreateSession(receiptId string, preKeyBundle *prekey.Bundle) error {

	signalAddress := protocol.NewSignalAddress(receiptId, 0)
	builder := session.NewBuilder(
		m.SessionStore, m.PreKeyStore, m.SignedPreKeyStore, m.IdentityStore, signalAddress, m.Serialize)
	return builder.ProcessBundle(preKeyBundle)
}

// CreateGroupSession
func (m *Manager) CreateGroupSession(groupId, participantId string) (*protocol.SenderKeyDistributionMessage, error) {
	signalAddress := protocol.NewSignalAddress(participantId, 0)
	senderKeyName := protocol.NewSenderKeyName(groupId, signalAddress)
	return m.groupBuilder.Create(senderKeyName)
}

// ProcessGroupSession
func (m *Manager) ProcessGroupSession(groupId, participantId string, data []byte) {

	signalAddress := protocol.NewSignalAddress(participantId, 0)
	senderKeyName := protocol.NewSenderKeyName(groupId, signalAddress)

	/*byte[][] messageParts = ByteUtil.split(serialized, 1, serialized.length - 1);
	byte     version      = messageParts[0][0];
	byte[]   message      = messageParts[1];*/
	// 需要将数据序列化
	senderKeyDistributionSerializer := &serializer.ProtoSenderKeyDistributionMessageSerializer{}
	senderKeyDistributionMessage, err := protocol.NewSenderKeyDistributionMessageFromBytes(data, senderKeyDistributionSerializer)
	if err != nil {
		log.Println("NewSenderKeyDistributionMessageFromBytes err :", err.Error())
		return
	}
	m.groupBuilder.Process(senderKeyName, senderKeyDistributionMessage)
}

// getGroupSessionCipher
func (m *Manager) getGroupSessionCipher(groupId, participantId string) *groups.GroupCipher {

	signalAddress := protocol.NewSignalAddress(participantId, 0)
	senderKeyName := protocol.NewSenderKeyName(groupId, signalAddress)
	// groupSession map
	if cipher, ok := m.groupSession[senderKeyName.String()]; ok {
		return cipher
	}

	cipher := groups.NewGroupCipher(m.groupBuilder, senderKeyName, m.SenderKeyStore)
	m.groupSession[senderKeyName.String()] = cipher
	return cipher
}
func (m *Manager) getSessionBuilder(id string) *session.Builder {
	signalAddress := protocol.NewSignalAddress(id, 0)
	sessionBuilder := session.NewBuilder(
		m.SessionStore, m.PreKeyStore, m.SignedPreKeyStore, m.IdentityStore, signalAddress, m.Serialize)
	return sessionBuilder
}

// getSessionCipher
func (m *Manager) getSessionCipher(id string) *session.Cipher {
	if cipher, ok := m.sessionCiphers[id]; ok {
		return cipher
	}

	if strings.Contains(id, "@") {
		id = strings.Split(id, "@")[0]
	}

	signalAddress := protocol.NewSignalAddress(id, 0)
	// create new Cipher
	newCipher := session.NewCipher(m.getSessionBuilder(id), signalAddress)
	//save newCipher to list
	m.sessionCiphers[id] = newCipher
	return newCipher
}

// groupEncrypt
func (m *Manager) groupEncrypt(id, participantId string, d []byte) (protocol.CiphertextMessage, error) {
	cipher := m.getGroupSessionCipher(id, participantId)
	if cipher == nil {
		return nil, errors.New("create cipher error")
	}

	log.Println(" cipher.Encrypt", hex.EncodeToString(d))
	return cipher.Encrypt(d)
}

// uEncrypt
func (m *Manager) uEncrypt(id string, d []byte) (protocol.CiphertextMessage, error) {
	cipher := m.getSessionCipher(id)
	if cipher == nil {
		return nil, errors.New("create cipher error")
	}

	log.Println(" cipher.Encrypt", hex.EncodeToString(d))
	return cipher.Encrypt(d)
}

// Encrypt
func (m *Manager) Encrypt(id string, d []byte, isGroup bool, participantId ...string) (protocol.CiphertextMessage, error) {
	//// 在进行 Proto 加密前 随机生成 0 ~ 16 Byte 进行数据填充
	randomPaddingData := randomPadding()
	// append randomPaddingData
	d = append(d, randomPaddingData...)

	//if is group
	if isGroup && len(participantId) > 0 && participantId[0] != "" { //
		return m.groupEncrypt(id, participantId[0], d)
	} else {
		return m.uEncrypt(id, d)
	}
}

func (m *Manager) decryptPreKeySignalMessageNew(from, participant string, d []byte) ([]byte, error) {
	var (
		c *session.Cipher
	)
	// id
	if participant != "" {
		from = participant
	}

	c = m.getSessionCipher(from)
	if c == nil {
		return nil, errors.New("get cipher error is null")
	}

	//get session builder
	sessionBuilder := m.getSessionBuilder(from)
	// serializer message
	signalMessageSerializer := &serializer.ProtoSignalMessageSerializer{}
	PreKeySignalMessageSerializer := &serializer.ProtoPreKeySignalMessageSerializer{}
	preKeySignalMessage, err := protocol.NewPreKeySignalMessageFromBytes(d, PreKeySignalMessageSerializer, signalMessageSerializer)
	if err != nil {
		return nil, err
	}
	// process
	_, err = sessionBuilder.Process(preKeySignalMessage)
	if err != nil {
		log.Println("decryptPreKeySignalMessageNew error", err)
	}
	//log.Println("decryptPreKeySignalMessageNew unsignedPreKeyID:",unsignedPreKeyID.Value,err)

	// decrypt
	decryptedData, err := c.Decrypt(preKeySignalMessage.WhisperMessage())
	return removePadding(decryptedData), err
}

//decryptPreKeySignalMessage
func (m *Manager) decryptPreKeySignalMessage(from, participant string, d []byte) ([]byte, error) {
	var (
		c *session.Cipher
	)
	// id
	if participant != "" {
		from = participant
	}

	c = m.getSessionCipher(from)
	if c == nil {
		return nil, errors.New("get cipher error is null")
	}
	// decrypt
	signalMessageSerializer := &serializer.ProtoSignalMessageSerializer{}
	PreKeysignalMessageSerializer := &serializer.ProtoPreKeySignalMessageSerializer{}
	preKeySignalMessage, err := protocol.NewPreKeySignalMessageFromBytes(d, PreKeysignalMessageSerializer, signalMessageSerializer)
	if err != nil {
		return nil, err
	}
	decryptedData, err := c.Decrypt(preKeySignalMessage.WhisperMessage())
	return removePadding(decryptedData), err
}

// decryptSignalMessage
func (m *Manager) decryptSignalMessage(from, participant string, d []byte) ([]byte, error) {
	var (
		c *session.Cipher
	)
	// id
	if participant != "" {
		from = participant
	}

	c = m.getSessionCipher(from)
	if c == nil {
		return nil, errors.New("get cipher error is null")
	}
	// decrypt
	signalMessageSerializer := &serializer.ProtoSignalMessageSerializer{}
	signalMessage, err := protocol.NewSignalMessageFromBytes(d, signalMessageSerializer)
	if err != nil {
		return nil, err
	}
	decryptedData, err := c.Decrypt(signalMessage)
	return removePadding(decryptedData), err
}

// decryptSenderKeyMessage
func (m *Manager) decryptSenderKeyMessage(from, participant string, d []byte) ([]byte, error) {
	// get group cipher
	cipher := m.getGroupSessionCipher(from, participant)
	// Serializer
	senderKeyMessageSerializer := &serializer.ProtoSenderKeyMessageSerializer{}
	senderKeyMessage, err := protocol.NewSenderKeyMessageFromBytes(d, senderKeyMessageSerializer)
	if err != nil {
		return nil, err
	}
	decryptedData, err := cipher.Decrypt(senderKeyMessage)
	return removePadding(decryptedData), err
}

// Decrypt decryptMsg
func (m *Manager) Decrypt(from, participant string, d []byte, deType string) ([]byte, error) {
	// decrypt
	switch deType {
	case "msg":
		return m.decryptSignalMessage(from, participant, d)
	case "pkmsg":
		return m.decryptPreKeySignalMessageNew(from, participant, d)
	case "skmsg":
		return m.decryptSenderKeyMessage(from, participant, d)
	case "frskmsg":

	}
	return nil, nil
}

// randomPadding 随机填充
func randomPadding() []byte {
	intn := rand.Intn(16)
	ending := make([]byte, intn, byte(intn))
	for i, _ := range ending {
		ending[i] = byte(intn)
	}
	for len(ending) == 0 {
		ending = randomPadding()
	}
	return ending
}

// RemovePadding 删除末尾填充
func removePadding(d []byte) []byte {
	if d == nil || len(d) <= 0 {
		return []byte{}
	}
	endByte := d[len(d)-1]
	checkData := d[len(d)-int(endByte):]
	// verify data consistency
	for _, datum := range checkData {
		if endByte != datum {
			return d
		}
	}
	return d[:len(d)-int(endByte)]
}
