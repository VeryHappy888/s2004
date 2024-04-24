package register

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/gogf/guuid"
	"log"
	"math/rand"
	"strings"
	"time"
	"ws-go/libsignal/ecc"
	"ws-go/libsignal/keys/identity"
	"ws-go/libsignal/serialize"
	"ws-go/libsignal/state/record"
	"ws-go/libsignal/util/keyhelper"
)

const WaConfigLogTag = "WAConfig"

type WAConfig struct {
	// 区号
	CC string
	// 手机号
	In string
	// guuid
	FDid string
	// 随机16位
	Exid string
	//
	MCC string
	// mnc
	MNC         string
	SimMcc      string
	SimMnc      string
	BackupToken string
	Id          string

	RegistrationId      uint32
	ClientStaticKeyPair *ecc.ECKeyPair
	IdentityKeyPair     *identity.KeyPair
	SignedPreKey        *record.SignedPreKey
	//新加给值
	EIdent         string
	EIdentPrivate  [32]byte
	EsKeyId        string
	EsKeyVal       string
	EsKeySig       string
	AuthKey        string
	StaticPubKey   string
	StaticPriKey   string
	IdentityPriKey string
	IdentityPubKey string
}

// SetRegistrationVal 注册给值
func (c *WAConfig) SetRegistrationVal() *WAConfig {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("SetRegistrationVal panic: %v\n", r)
		}
	}()
	//e_ident
	c.EIdent = base64.RawURLEncoding.EncodeToString(c.IdentityKeyPair.PublicKey().Serialize()[1:])
	c.EIdentPrivate = c.IdentityKeyPair.PrivateKey().Serialize()
	eSkeyId := make([]byte, 4)
	binary.BigEndian.PutUint32(eSkeyId, c.SignedPreKey.ID())
	c.EsKeyId = base64.RawURLEncoding.EncodeToString(eSkeyId[1:])
	c.EsKeyVal = base64.RawURLEncoding.EncodeToString(c.SignedPreKey.KeyPair().PublicKey().Serialize()[1:])
	eSkeySig := make([]byte, 64)
	signature := c.SignedPreKey.Signature()
	copy(eSkeySig[:], signature[:])
	c.EsKeySig = base64.RawURLEncoding.EncodeToString(eSkeySig)
	c.AuthKey = base64.RawURLEncoding.EncodeToString(c.ClientStaticKeyPair.PublicKey().Serialize()[1:])
	//add
	clientStaticpriKey := c.ClientStaticKeyPair.PrivateKey().Serialize()
	_clientStaticPriKey := make([]byte, len(clientStaticpriKey))
	copy(_clientStaticPriKey[:], clientStaticpriKey[:])
	cliStaticKeyPair := append(_clientStaticPriKey, c.ClientStaticKeyPair.PublicKey().Serialize()[1:]...)
	c.StaticPubKey = base64.StdEncoding.EncodeToString(cliStaticKeyPair[32:])
	c.StaticPriKey = base64.StdEncoding.EncodeToString(cliStaticKeyPair[:32])
	_eIdentPrivate := make([]byte, len(c.EIdentPrivate))
	copy(_eIdentPrivate[:], c.EIdentPrivate[:])
	c.IdentityPriKey = base64.StdEncoding.EncodeToString(_eIdentPrivate)
	c.IdentityPubKey = c.EIdent
	return c
}

func (c *WAConfig) GenConfigJson(edgeRouting string) interface{} {
	/*clientStaticpriKey := c.ClientStaticKeyPair.PrivateKey().Serialize()
	_clientStaticPriKey := make([]byte, len(clienticpriKey))Stat
	copy(_clientStaticPriKey[:], clientStaticpriKey[:])
	cliStaticKeyPair := append(_clientStaticPriKey, c.ClientStaticKeyPair.PublicKey().Serialize()[1:]...)
	*/
	//number := rand.Intn(5)
	mcc := "460"
	mnc := "000"
	/*if number == 1 {
		mcc = "310"
		mnc = "070"
	} else if number == 2 {
		mnc = "02"
	} else if number == 3 {
		mnc = "03"
	} else if number == 4 {
		mnc = "05"
	} else if number == 5 {
		mnc = "00"
	}*/
	config := map[string]string{
		"status": "ok",
		"cc":     c.CC,
		//"client_static_keypair": base64.StdEncoding.EncodeToString(cliStaticKeyPair),
		"edge_routing_info": edgeRouting,
		//"expid":                 c.Exid,
		"phone_id":       c.FDid,
		"id":             c.Id,
		"login":          c.In,
		"mcc":            mcc,
		"mnc":            mnc,
		"phone":          c.In,
		"pushname":       randSeq(7),
		"sim_mcc":        "0",
		"sim_mnc":        "234",
		"StaticPubKey":   c.StaticPubKey,
		"StaticPriKey":   c.StaticPriKey,
		"IdentityPriKey": c.IdentityPriKey,
		"IdentityPubKey": c.IdentityPubKey,
	}
	//d, _ := json.Marshal(config)
	return config
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(62)]
	}
	return string(b)
}

// GenerateWAConfig 生成随机设备配置
func GenerateWAConfig(iso string) *WAConfig {
	var err error
	config := &WAConfig{}
	config.SimMnc = "234"
	config.SimMcc = "012"

	config.MCC = "234"
	config.MNC = "011"

	if iso != "" {
		isp := ISP(iso)
		config.SimMnc = isp.MNC
		config.SimMcc = isp.MCC
		config.MCC = isp.MCC
		config.MNC = isp.MNC
	}

	config.FDid = strings.ToUpper(guuid.New().String())

	exidData := []byte(strings.ReplaceAll(guuid.New().String(), "-", "")[:16])
	config.Exid = base64.StdEncoding.EncodeToString(exidData)

	config.RegistrationId = keyhelper.GenerateRegistrationID()

	config.ClientStaticKeyPair, err = ecc.GenerateKeyPair()
	if err != nil {
		fmt.Errorf(err.Error())
		return nil
	}
	config.IdentityKeyPair, err = keyhelper.GenerateIdentityKeyPair()
	if err != nil {
		fmt.Errorf(err.Error())
		return nil
	}

	config.SignedPreKey, err = keyhelper.GenerateSignedPreKey(config.IdentityKeyPair, uint32(rand.Int()), &serialize.JSONSignedPreKeyRecordSerializer{})
	if err != nil {
		fmt.Errorf(err.Error())
		return nil
	}
	idData := []byte(strings.ReplaceAll(guuid.New().String(), "-", "")[0:20])
	config.Id = base64.StdEncoding.EncodeToString(idData)
	tokenData := []byte(strings.ReplaceAll(guuid.New().String(), "-", "")[0:15])
	config.BackupToken = base64.StdEncoding.EncodeToString(tokenData)
	return config
}

func printStringWithSymbol(input string, interval int, symbol string) string {
	count := 1
	txt := ""
	for _, char := range input {
		count++
		if count%interval == 0 && count/interval < len(input) {
			txt += symbol
		}
		txt += string(char)
	}
	return txt
}

func RandID(c int) string {
	b := make([]byte, c)
	crand.Read(b)
	var ss []string
	for _, v := range b {
		ss = append(ss, fmt.Sprintf("%%%02X", v))
	}
	return strings.Join(ss, "")
}
