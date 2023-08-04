package register

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogf/guuid"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"ws-go/libsignal/ecc"
	"ws-go/protocol/utils"
	"ws-go/protocol/waproto"
)

type VerifyMethod string

const (
	SMS   VerifyMethod = "sms"
	VOICE VerifyMethod = "voice"
)

var WAPUBKEY, _ = ecc.DecodePoint([]byte{
	5, 142, 140, 15, 116, 195, 235, 197, 215, 166, 134, 92, 108,
	60, 132, 56, 86, 176, 97, 33, 204, 232, 234, 119, 77, 34, 251,
	111, 18, 37, 18, 48, 45,
}, 0)

const waRegistrationLogTag = "WARegistration"

type WaRegistration struct {
	Lg        string //en
	Lc        string //US
	WAId      string
	Proxy     string
	DeEnv     Env
	DeConfig  *WAConfig
	CachePath string
}

func NewWaRegistration() *WaRegistration {
	return &WaRegistration{}
}

func (r *WaRegistration) createReqTask(api string, params *url.Values) *WaRequestTask {
	p, err := url.Parse(r.Proxy)
	if err != nil {
		panic(errors.New("proxy fail"))
	}
	task := &WaRequestTask{
		HttpProxy:  p,
		Api:        api,
		Parameter:  &url.Values{},
		HttpMethod: http.MethodGet,
		Header:     DefaultWAHeader(r.DeEnv),
		OptionMethod: func(task *WaRequestTask) error {
			keyPair, _ := ecc.GenerateKeyPair()
			enAfterData, err := r.EncodeParams(params.Encode(), keyPair)
			if err != nil {
				return err
			}
			//task.setReqData(enAfterData)
			reqData := append(keyPair.PublicKey().Serialize()[1:], enAfterData...)
			task.Parameter.Add("ENC", base64.RawStdEncoding.EncodeToString(reqData))
			return nil
		},
	}
	return task
}

func (r *WaRegistration) createReqTask2(api string, params string) *WaRequestTask {
	p, err := url.Parse(r.Proxy)
	if err != nil {
		panic(errors.New("proxy fail"))
	}
	task := &WaRequestTask{
		HttpProxy:  p,
		Api:        api,
		Parameter:  &url.Values{},
		HttpMethod: http.MethodGet,
		Header:     DefaultWAHeader(r.DeEnv),
		OptionMethod: func(task *WaRequestTask) error {
			keyPair, _ := ecc.GenerateKeyPair()
			enAfterData, err := r.EncodeParams(params, keyPair)
			if err != nil {
				return err
			}
			//task.setReqData(enAfterData)
			reqData := append(keyPair.PublicKey().Serialize()[1:], enAfterData...)
			task.Parameter.Add("ENC", base64.RawStdEncoding.EncodeToString(reqData))
			return nil
		},
	}
	return task
}

func (r *WaRegistration) ExistsRequest(cc, phone string) (result *ExistsResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}

	Token, err := r.DeEnv.GetToken(phone)

	if err != nil {
		return nil, err
	}

	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Del("offline_ab")
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		params.Add("token", Token)
		params.Add("offline_ab", "{\"exposure\":[\"registration_offline_universe_release|funnel_logging|test\"],\"metrics\":{}}")
	} else {
		params.Del("read_phone_permission_granted")
		params.Add("offline_ab", `{"exposure":["dummy_aa_offline_rid_universe_ios|dummy_aa_offline_rid_experiment_ios|test"],"metrics":{}}`)
	}
	// 创建WA请求任务
	fmt.Println("查询号码状态", params.Encode())
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/exist?", params).Execute()
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(bytes, &result); err != nil {
		return result, err
	}
	return result, err
}

func (r *WaRegistration) BusinessExistRequest(cc, phone string) (result *ExistsResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}

	Token, err := r.DeEnv.GetToken(phone)
	if err != nil {
		return nil, err
	}

	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Add("token", Token)
	params.Del("offline_ab")
	params.Add("offline_ab", "{\"exposure\":[],\"metrics\":{}}")
	// 创建WA请求任务
	fmt.Println("查询Business号码状态", params.Encode())
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/exist?", params).Execute()
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(bytes, &result); err != nil {
		return result, err
	}
	return result, err
}

// FinishRegistration  效验验证码
func (r *WaRegistration) FinishRegistration(cc, phone, code string) (result *RegisterResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}
	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Del("read_phone_permission_granted")
	params.Del("offline_ab")
	params.Del("sim_state")
	params.Del("client_metrics")
	isp := ISP(cc)
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		params.Add("client_metrics", `{"attempts":1}`)
		params.Add("sim_mcc", isp.MCC)
		params.Add("sim_mnc", isp.MNC)
		params.Add("mnc", "000")
		params.Add("mcc", isp.MCC)
	}
	params.Add("entered", "1")
	params.Add("code", code)
	fmt.Println("注册请求验证码->", params.Encode())
	// 创建WA请求任务
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/register?", params).Execute()
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(bytes, &result); err != nil {
		return result, err
	}
	//clientLog
	if result.EdgeRoutingInfo != "" {
		// go1
		funnelId := guuid.New().String()
		go r.clientLog(cc, phone, "successful", "verify_sms", funnelId, "google_backup")
		go r.clientLog(cc, phone, "next", "no_backup_found", funnelId, "profile_photo")
		go r.clientLog(cc, phone, "not_now", "google_backup", funnelId, "no_backup_found")
		go r.clientLog(cc, phone, "no_tap", "profile_photo", funnelId, "home")
	}
	return result, err
}

// FinishBusinessRegistration  商业版效验验证码
func (r *WaRegistration) FinishBusinessRegistration(cc, phone, code string) (result *RegisterResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}
	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Del("read_phone_permission_granted")
	params.Del("offline_ab")
	params.Del("sim_state")
	params.Del("client_metrics")
	params.Add("client_metrics", `{"attempts":1}`)
	params.Add("sim_mcc", "234")
	params.Add("entered", "1")
	params.Add("mnc", "000")
	params.Add("sim_mnc", "000")
	params.Add("mcc", "460")
	params.Add("code", code)
	randId, _ := strconv.ParseUint(utils.MakeYearDaysRand(19), 0, 64)
	fmt.Println(randId)
	vName := waproto.VerifiedName{
		VerifiedTow: nil,
		VerifiedOne: &waproto.VerifiedName_VerifiedOne{
			VerifiedOne1: proto.Uint64(uint64(randId)),
			VerifiedOne2: proto.String("smb:wa"),
			VerifiedOne4: proto.String(""),
		},
	}
	var random [64]byte
	ran := rand.Reader
	io.ReadFull(ran, random[:])
	message, _ := proto.Marshal(vName.VerifiedOne)
	byteSign := ecc.Sign(&r.DeConfig.EIdentPrivate, message, random)
	vName.VerifiedTow = byteSign[:]
	byteVName, _ := proto.Marshal(&vName)
	params.Add("vname", base64.RawStdEncoding.EncodeToString(byteVName))
	fmt.Println("注册请求验证码->", params.Encode())
	// 创建WA请求任务
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/register?", params).Execute()
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(bytes, &result); err != nil {
		return result, err
	}
	return result, err
}

// CalculateSignature signs a message with the given private key.
/*func CalculateSignature(signingKey ECPrivateKeyable, message []byte) [64]byte {
	logger.Debug("Signing bytes with signing key")
	// Get cryptographically secure random numbers.
	var random [64]byte
	r := rand.Reader
	io.ReadFull(r, random[:])

	// Get the private key.
	privateKey := signingKey.Serialize()

	// Sign the message.
	signature := sign(&privateKey, message, random)
	return *signature
}*/

// RequestVerifyCode 请求验证码
func (r *WaRegistration) RequestVerifyCode(cc, phone string, method VerifyMethod) (result *SendVerifyCodeResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}

	Token, err := r.DeEnv.GetToken(phone)

	if err != nil {
		return nil, err
	}

	r.DeConfig.In = phone
	// 生成默认请求参数
	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Del("network_operator_name")
	params.Del("read_phone_permission_granted")
	params.Del("offline_ab")
	params.Del("sim_state")
	params.Del("sim_operator_name") //hasav
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		params.Add("mcc", r.DeConfig.MCC)
		params.Add("mnc", r.DeConfig.MNC)
	}
	params.Add("sim_mcc", r.DeConfig.SimMcc)
	params.Add("sim_mnc", r.DeConfig.SimMnc)
	params.Add("method", string(method))
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		params.Add("reason", "")
	}
	params.Add("cellular_strengt", "2")
	params.Add("token", Token)
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		params.Del("client_metrics")
		params.Add("client_metrics", "{\"attempts\":1}")
		params.Add("hasav", "2")
	}

	var bytes []byte

	if r.DeEnv.EnvInfo().PLATFORM == "Apple" {
		T := params.Encode()
		T += fmt.Sprintf("&id=%v", RandID(16))
		// 创建WA请求任务
		fmt.Println("请求验证码", T)
		bytes, err = r.createReqTask2("https://v.whatsapp.net/v2/code?", T).Execute()
	} else {
		// 创建WA请求任务
		fmt.Println("请求验证码", params.Encode())
		bytes, err = r.createReqTask("https://v.whatsapp.net/v2/code?", params).Execute()
	}

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// RequestBusinessVerifyCode 请求验证码
func (r *WaRegistration) RequestBusinessVerifyCode(cc, phone string, method VerifyMethod) (result *SendVerifyCodeResult, err error) {
	if len(cc) == 0 || len(phone) == 0 {
		return
	}

	Token, err := r.DeEnv.GetToken(phone)
	if err != nil {
		return nil, err
	}

	r.DeConfig.In = phone
	// 生成默认请求参数
	params := GenWARegistrationParams(cc, phone, r.DeConfig, r)
	params.Del("network_operator_name")
	params.Del("read_phone_permission_granted")
	params.Del("offline_ab")
	params.Del("sim_state")
	params.Del("sim_operator_name") //hasav
	params.Add("mcc", r.DeConfig.MCC)
	params.Add("mnc", r.DeConfig.MNC)
	params.Add("sim_mcc", r.DeConfig.SimMcc)
	params.Add("sim_mnc", r.DeConfig.SimMnc)
	params.Add("method", string(method))
	params.Add("reason", "")
	params.Add("token", Token)
	params.Del("client_metrics")
	params.Add("client_metrics", "{\"attempts\":1}")
	params.Add("hasav", "2")
	fmt.Println("请求验证码", params.Encode())
	// 创建WA请求任务
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/code?", params).Execute()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// ExistsResult 查询是否存在返回结果
type ExistsResult struct {
	//状态：ok
	Status string `json:"status"`
	//原因
	Reason string `json:"reason"`
	//登录账号
	Login string `json:"login"`

	SmsLength uint32 `json:"sms_length"`

	VoiceLength uint32 `json:"voice_length"`

	SmsWait uint32 `json:"sms_wait"`

	VoiceWait uint32 `json:"voice_wait"`

	FlashType uint32 `json:"flash_type"`
}

// 注册返回结果
// {"status":"ok","login":"8613637844373","type":"new","edge_routing_info":"CAgIBQ==","security_code_set":false}
// {"login":"6283811943518","status":"fail","reason":"missing"}
type RegisterResult struct {
	//状态：ok
	Status string `json:"status"`
	//原因
	Reason string `json:"reason"`
	//登录账号
	Login string `json:"login"`
	//类型，新账号
	Type string `json:"type"`
	//边缘路由信息
	EdgeRoutingInfo string `json:"edge_routing_info"`
	//安全码设置
	SecurityCodeSet bool `json:"security_code_set"`
}

type SendVerifyCodeResult struct {
	//状态：sent
	Status string `json:"status"`
	//原因
	Reason string `json:"reason"`
	//登录账号
	Login string `json:"login"`
	//通知
	NotifyAfter float64 `json:"notify_after"`
	//长度
	Length float64 `json:"length"`
	//方法：sms
	Method string `json:"method"`
	//复述
	RetryAfter float64 `json:"retry_after"`
	//短信等待
	SmsWait float64 `json:"sms_wait"`
	//语音等待
	VoiceWait float64 `json:"voice_wait"`
	//闪存类型：0为不存
	FlashType float64 `json:"flash_type"`
}

func (w *WaRegistration) EncodeParams(data string, keyPair *ecc.ECKeyPair) ([]byte, error) {
	secret := ecc.GenerateSharedSecret(keyPair.PrivateKey().Serialize(), WAPUBKEY.PublicKey())
	enAfter, err := ecc.AesGcmEncrypt(secret, make([]byte, 12), []byte{}, []byte(data))
	if err != nil {
		return nil, err
	}
	return enAfter, nil
}

// DefaultWARegistrationParams WhatsApp 注册共有参数
func GenWARegistrationParams(cc string, phone string, deConfig *WAConfig, r *WaRegistration) *url.Values {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("GenWARegistrationParams panic: %v\n", r)
		}
	}()
	if deConfig == nil {
		deConfig = GenerateWAConfig(r.Lc)
		deConfig.CC = cc
		deConfig.In = phone
	}

	parmas := url.Values{}
	//parmas.Add("backup_token",deConfig.BackupToken)
	parmas.Add("read_phone_permission_granted", "0")
	parmas.Add("cc", cc)
	parmas.Add("in", phone)
	if r.Lc != "" {
		parmas.Add("lc", r.Lc)
	} else {
		parmas.Add("lc", "CN")
	}
	if r.Lg != "" {
		parmas.Add("lg", r.Lg)
	} else {
		parmas.Add("lg", "zh")
	}
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		parmas.Add("mistyped", "7")
	}
	parmas.Add("offline_ab", "{\"exposure\":[],\"metrics\":{}}")

	eRegId := make([]byte, 4)
	binary.BigEndian.PutUint32(eRegId, deConfig.RegistrationId)
	parmas.Add("e_regid", base64.RawStdEncoding.EncodeToString(eRegId))

	parmas.Add("e_keytype", base64.RawStdEncoding.EncodeToString([]byte{ecc.DjbType}))
	parmas.Add("e_ident", deConfig.EIdent)
	if deConfig.EIdent == "" {
		parmas.Del("e_ident")
		parmas.Add("e_ident", base64.RawStdEncoding.EncodeToString(deConfig.IdentityKeyPair.PublicKey().Serialize()[1:]))
	}

	eSkeyId := make([]byte, 4)
	binary.BigEndian.PutUint32(eSkeyId, deConfig.SignedPreKey.ID())
	parmas.Add("e_skey_id", deConfig.EsKeyId)
	if deConfig.EsKeyId == "" {
		parmas.Del("e_skey_id")
		parmas.Add("e_skey_id", base64.RawStdEncoding.EncodeToString(eSkeyId[1:]))
	}

	parmas.Add("e_skey_val", deConfig.EsKeyVal)
	if deConfig.EsKeyVal == "" {
		parmas.Del("e_skey_val")
		parmas.Add("e_skey_val", base64.RawStdEncoding.EncodeToString(deConfig.SignedPreKey.KeyPair().PublicKey().Serialize()[1:]))
	}
	parmas.Add("e_skey_sig", deConfig.EsKeySig)
	if deConfig.EsKeySig == "" {
		eSkeySig := make([]byte, 64)
		signature := deConfig.SignedPreKey.Signature()
		copy(eSkeySig[:], signature[:])
		parmas.Del("e_skey_sig")
		parmas.Add("e_skey_sig", base64.RawStdEncoding.EncodeToString(eSkeySig))
	}

	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		//backup_token
		parmas.Add("backup_token", deConfig.BackupToken)
	}
	parmas.Add("fdid", deConfig.FDid)
	parmas.Add("expid", deConfig.Exid)

	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		parmas.Add("network_radio_type", "1")
		parmas.Add("simnum", "0")
		parmas.Add("hasinrc", "1")
		parmas.Add("sim_state", "5")
		parmas.Add("client_metrics", "{\"attempts\":1,\"was_activated_from_stub\":false}") //attempts请求次数
		parmas.Add("network_operator_name", "CHINA MOBILE")
		parmas.Add("sim_operator_name", "giffgaff")
		pid := strconv.FormatInt(utils.RangeRand(18000, 18999), 10)
		parmas.Add("pid", pid)
	}
	parmas.Add("rc", "0")
	if r.DeEnv.EnvInfo().PLATFORM == "android" {
		parmas.Add("id", deConfig.Id)
	}

	if deConfig.AuthKey == "" {
		authkey := base64.RawStdEncoding.EncodeToString(deConfig.ClientStaticKeyPair.PublicKey().Serialize()[1:])
		parmas.Del("authkey")
		parmas.Add("authkey", url.QueryEscape(authkey))
	} else {
		parmas.Add("authkey", url.QueryEscape(deConfig.AuthKey))
	}
	return &parmas
}

// DefaultWAHeader HTTP请求协议头
func DefaultWAHeader(deEnv Env) http.Header {
	header := http.Header{}
	header.Add("User-Agent", deEnv.WAUserAgent())
	header.Add("WaMsysRequest", "1")
	header.Add("request_token", guuid.New().String()) //验证请求token是否每次请求不一样问题 1 2
	header.Add("Accept", "'text/json'")
	header.Add("Content-Type", "application/x-www-form-urlencoded")
	return header
}

// clientLog 注册client_log
func (r *WaRegistration) clientLog(cc, phone, actionTaken, previousScreen, funnelId, currentScreen string) (result *RegisterResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("clientLog panic: %v\n", r)
		}
	}()
	if len(cc) == 0 || len(phone) == 0 {
		return
	}
	if r.DeConfig == nil {
		r.DeConfig = GenerateWAConfig(r.Lc)
		r.DeConfig.CC = cc
		r.DeConfig.In = phone
	}
	params := url.Values{}
	params.Add("cc", cc)
	params.Add("in", phone)
	if r.Lc != "" {
		params.Add("lc", r.Lc)
	} else {
		params.Add("lc", "CN")
	}
	if r.Lg != "" {
		params.Add("lg", r.Lg)
	} else {
		params.Add("lg", "zh")
	}
	params.Add("backup_token", r.DeConfig.BackupToken)
	params.Add("id", r.DeConfig.Id)
	eRegId := make([]byte, 4)
	binary.BigEndian.PutUint32(eRegId, r.DeConfig.RegistrationId)
	params.Add("e_regid", base64.RawStdEncoding.EncodeToString(eRegId))
	params.Add("e_skey_sig", r.DeConfig.EsKeySig)
	if r.DeConfig.EsKeySig == "" {
		eSkeySig := make([]byte, 64)
		signature := r.DeConfig.SignedPreKey.Signature()
		copy(eSkeySig[:], signature[:])
		params.Del("e_skey_sig")
		params.Add("e_skey_sig", base64.RawStdEncoding.EncodeToString(eSkeySig))
	}
	params.Add("action_taken", actionTaken)
	params.Add("expid", r.DeConfig.Exid)
	params.Add("e_ident", r.DeConfig.EIdent)
	if r.DeConfig.EIdent == "" {
		params.Del("e_ident")
		params.Add("e_ident", base64.RawStdEncoding.EncodeToString(r.DeConfig.IdentityKeyPair.PublicKey().Serialize()[1:]))
	}
	params.Add("previous_screen", previousScreen)
	eSkeyId := make([]byte, 4)
	binary.BigEndian.PutUint32(eSkeyId, r.DeConfig.SignedPreKey.ID())
	params.Add("e_skey_id", r.DeConfig.EsKeyId)
	if r.DeConfig.EsKeyId == "" {
		params.Del("e_skey_id")
		params.Add("e_skey_id", base64.RawStdEncoding.EncodeToString(eSkeyId[1:]))
	}
	params.Add("fdid", r.DeConfig.FDid)
	params.Add("funnel_id", funnelId)
	params.Add("e_skey_val", r.DeConfig.EsKeyVal)
	if r.DeConfig.EsKeyVal == "" {
		params.Del("e_skey_val")
		params.Add("e_skey_val", base64.RawStdEncoding.EncodeToString(r.DeConfig.SignedPreKey.KeyPair().PublicKey().Serialize()[1:]))
	}
	params.Add("e_keytype", base64.RawStdEncoding.EncodeToString([]byte{ecc.DjbType}))
	params.Add("current_screen", currentScreen)
	if r.DeConfig.AuthKey == "" {
		authkey := base64.RawStdEncoding.EncodeToString(r.DeConfig.ClientStaticKeyPair.PublicKey().Serialize()[1:])
		params.Del("authkey")
		params.Add("authkey", url.QueryEscape(authkey))
	} else {
		params.Add("authkey", url.QueryEscape(r.DeConfig.AuthKey))
	}
	fmt.Println("client_log-req>", params.Encode())
	// 创建WA请求任务
	bytes, err := r.createReqTask("https://v.whatsapp.net/v2/client_log?", &params).Execute()
	if err != nil {
		return result, err
	}
	fmt.Println("client_log-resp-->", string(bytes))
	/*if err = json.Unmarshal(bytes, &result); err != nil {
		return result, err
	}*/
	return nil, err
}
