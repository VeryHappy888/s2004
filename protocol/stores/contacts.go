package stores

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/database/gdb"
	"os"
	"ws-go/protocol/define"

	_ "github.com/mattn/go-sqlite3"
)

// ContactStores
type ContactStores struct {
	dbSource gdb.DB
	UserName string
}

// createContactTables create contact tables
func (c *ContactStores) createContactTables() {
	// create contacts
	_, err := c.dbSource.Exec(`CREATE TABLE "wa_contacts" (
	"_id"	INTEGER PRIMARY KEY AUTOINCREMENT,
	"jid"	TEXT NOT NULL,
	"is_whatsapp_user"	BOOLEAN NOT NULL,
	"status"	TEXT,
	"status_timestamp"	INTEGER,
	"number"	TEXT,
	"raw_contact_id"	INTEGER,
	"display_name"	TEXT,
	"phone_type"	INTEGER,
	"phone_label"	TEXT,
	"unseen_msg_count"	INTEGER,
	"photo_ts"	INTEGER,
	"thumb_ts"	INTEGER,
	"photo_id_timestamp"	INTEGER,
	"given_name"	TEXT,
	"family_name"	TEXT,
	"wa_name"	TEXT,
	"sort_name"	TEXT,
	"nickname"	TEXT,
	"company"	TEXT,
	"title"	TEXT,
	"status_autodownload_disabled"	INTEGER,
	"keep_timestamp"	INTEGER,
	"is_spam_reported"	INTEGER,
	"is_sidelist_synced"	BOOLEAN DEFAULT 0,
	"is_business_synced"	BOOLEAN DEFAULT 0
);`)
	if err != nil {
		panic(err)
	}
	// group participants
	_, err = c.dbSource.Exec(`
		CREATE TABLE "group_participants" (
		"_id"	INTEGER PRIMARY KEY AUTOINCREMENT,
		"gjid"	TEXT NOT NULL,
		"jid"	TEXT NOT NULL,
		"admin"	INTEGER,
		"pending"	INTEGER,
		"sent_sender_key"	INTEGER);
	`)
	if err != nil {
		panic(err)
	}
}

// check check contact data bases  is exist
func (c *ContactStores) check() bool {
	dbPath := fmt.Sprintf("%s/%s/contacts", define.DefaultDbPath, c.UserName)
	// add data bases config
	gdb.AddConfigNode(c.UserName, gdb.ConfigNode{
		Type:    "sqlite",
		Charset: "utf8",
		Link:    dbPath,
	})

	// get data base connect
	if db, err := gdb.New(c.UserName); err != nil {
		return false
	} else {
		c.dbSource = db
	}

	// exist databases file
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// create tables
		c.createContactTables()
	}

	return true
}

// AddContact add contact to data base
func (c *ContactStores) AddContact(jid, name, status, number string) error {
	// use tables
	waContactModel := c.dbSource.Model("wa_contacts")
	mapData := gdb.Map{
		"jid":              jid,
		"status":           status,
		"number":           number,
		"display_name":     name,
		"is_whatsapp_user": 1,
	}
	_, err := waContactModel.Insert(mapData)
	return err
}

// AddGroupParticipants add group participant to data base
func (c *ContactStores) AddGroupParticipant(gjid, jid string, admin int) error {
	// use tables
	groupParticipantsModel := c.dbSource.Model("group_participants")
	anyMap := gdb.Map{
		"gjid":  gjid,
		"jid":   jid,
		"admin": admin,
	}
	_, err := groupParticipantsModel.Insert(anyMap)
	/**/
	return err
}

// GetAllContact get all contact
func (c *ContactStores) GetAllContact() (gdb.Result, error) {
	// use tables
	waContactModel := c.dbSource.Model("wa_contacts")
	return waContactModel.All()
}

// GetSentSenderKey  get group participant field sent_sender_key
func (c *ContactStores) GetSentSenderKey(gJid, jid string) (int, error) {
	// 群聊天中会为用户创建 sender key 如果创建成功并发送后 该置 为
	// use tables
	groupParticipantsModel := c.dbSource.Model("group_participants")
	result, err := groupParticipantsModel.
		Where("gjid=? AND jid =?", gJid, jid).
		FindOne()
	if err != nil {
		return -1, err
	}
	// get sent_sender_key
	if sentSenderKey, ok := result["sent_sender_key"]; ok {
		return sentSenderKey.Int(), nil
	}
	return -1, errors.New("not field sent_sender_key")
}

// UpdateGroupParticipant update sent_sender_key
func (c *ContactStores) UpdateGroupParticipantSentSenderKey(gJid, jid string, sentSenderKey int) error {
	// use tables
	groupParticipantsModel := c.dbSource.Model("group_participants")
	anyMap := gdb.Map{"sent_sender_key": sentSenderKey}
	_, err := groupParticipantsModel.
		Data(anyMap).
		Where("gjid=? AND jid =?", gJid, jid).
		Update()
	return err
}

// DelGroupParticipant del
func (c *ContactStores) DelGroupParticipant(gJid, jid string) error {
	// use tables
	groupParticipantsModel := c.dbSource.Model("group_participants")
	_, err := groupParticipantsModel.Delete("gjid=? AND jid =?", gJid, jid)
	return err
}

// NewContactStores
func NewContactStores(u string) *ContactStores {
	contactStores := &ContactStores{UserName: u}
	// check data bases
	contactStores.check()
	return contactStores
}
