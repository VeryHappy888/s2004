package test

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/gogf/gf/frame/g"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"ws-go/protocol/app"
	"ws-go/protocol/axolotl"
	"ws-go/protocol/stores"
	"ws-go/wslog"
)

func TestManager_GeneratingPreKeys(t *testing.T) {
	axolotlManager, _ := axolotl.NewAxolotlManager("test")
	axolotlManager.GeneratingPreKeys()

}

func TestNewContactStores(t *testing.T) {
	contactStores := stores.NewContactStores("test")
	t.Log(contactStores.AddContact("test1", "", "status", "1111111111111"))
	cs, err := contactStores.GetAllContact()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(cs.Json())
}

func TestNewWaAppCli(t *testing.T) {
	g.Cfg().SetFileName("config.toml")
	wslog.InitWsLogger()
	/*
	   {
	       "Socks5": "socks5://162.0.234.9:8088",
	       "AuthBody": {
	           "username": 14233767293,
	           "passive": false,
	           "user_agent": {
	               "platform": 0,
	               "app_version": {
	                   "primary": 2,
	                   "secondary": 20,
	                   "tertiary": 202,
	                   "quaternary": 24
	               },
	               "mcc": "000",
	               "mnc": "000",
	               "os_version": "7.1.2",
	               "manufacturer": "google",
	               "device": "sailfish",
	               "os_build_number": "aosp_sailfish-userdebug 7.1.2 N2G47O eng.root.20200802.022802 test-keys",
	               "phone_id": "90196710-7a70-45cf-8d8e-364c40d79296",
	               "locale_language_iso_639_1": "zh",
	               "locale_country_iso_3166_1_alpha_2": "CN"
	           },
	           "push_name": "Ffg",
	           "short_connect": false,
	           "connect_type": 1
	       },
	       "AuthHexData": "",
	       "StaticPriKey": "qCiLjgBvggI/BaCrEElSubDLq2Q/ehPk1LyZWQTFX0A=",
	       "StaticPubKey": "eDCkhPQb9hrEHA7ZKnhCc0ja4O5rHr+wwt6zvGGWOE8="
	   }

	*/
	priKey, _ := base64.StdEncoding.DecodeString("qPqJleEfh4/aFRt8KqmasVm29cg1BZ13aJMt9b9X8UvCv5/yBmRvUCsnwQQddKsFAliqObQMXam2EU56iSDtJw==")
	t.Log(base64.StdEncoding.EncodeToString(priKey[:32]), base64.StdEncoding.EncodeToString(priKey[32:]))
	//{"cc":"","client_static_keypair":"qCiLjgBvggI/BaCrEElSubDLq2Q/ehPk1LyZWQTFX0B4MKSE9Bv2GsQcDtkqeEJzSNrg7msev7DC3rO8YZY4Tw==","edge_routing_info":"CAsIDA==","expid":"NWYzOGIwMDExOGEyNDkwOA==","fdid":"90196710-7a70-45cf-8d8e-364c40d79296","id":"ZTI5MDdhYTAzNDcyNGE1YWIzMjY=","login":"4233767293","mcc":"460","mnc":"01","phone":"4233767293","pushname":"yowsup","server_static_public":"LKFtqKAASnxIgl89fmUjtpxpL8WVkwPZLMk3ybT3ggI=","sim_mcc":"000","sim_mnc":"000"}
	//{"cc":"","client_static_keypair":"mOKPCGExsuz6mpzqks3dWVf+56Dtz9VTGMpmkDXTamvToz1zsXuTLhHM0OYivd+Gp7W+vTIMIHEzoivBr6XBEg==","edge_routing_info":"CAsIDA==","expid":"ZWQxMmRmM2M5ODI0NGQ5NQ==","fdid":"d6cd4a85-7598-455d-be7d-fd9e8f44c90f","id":"YmU2ZGM4NDkxOGY4NDE3M2I5ODM=","login":"8329559489","mcc":"460","mnc":"01","phone":"8329559489","pushname":"yowsup","server_static_public":"LKFtqKAASnxIgl89fmUjtpxpL8WVkwPZLMk3ybT3ggI=","sim_mcc":"000","sim_mnc":"000"}

	//priKey, _ := base64.StdEncoding.DecodeString("UPcIqerhuaYnwovLI123HBLYDQLJY26XpDgBCyUpzWkIO4KZir0lGRFlZmPn/PIMrxj5ygX+FNsvpA50nzLXSw==")
	//priKey, _ := base64.StdEncoding.DecodeString("WDSQ7nMYl0WJuT6MqrATC4TwRCDY/fq4vKvSYdfX5EsYW33sQn/fQx/hhadZLVZUadxT5F+H+Mq3BSUbhA6bew==")
	//priKey, _ := base64.StdEncoding.DecodeString("cCUUuTsgaQL3smMWPdoRYbFT095bBBv0uY+5AOo2v0Oc4nPlWa6FXbpyjI8V7XodyKgoVFnpLG3Sez2XVYypNQ==")
	UAHex := "08e9d3d48af1b60118012ab101080012090802101418ca0120181a0330303022033030302a05372e312e323206676f6f676c653a087361696c666973684247616f73705f7361696c666973682d75736572646562756720372e312e32204e324734374f20656e672e726f6f742e32303230303830322e30323238303220746573742d6b6579734a2463356531616131642d643234662d343031382d383933332d6331386530396462633961645a027a686202434e6a087361696c666973683a034666674ddb0f94cb50006001"
	//UAHex := "08e8c982cf840d18012a73080012090802101418cd0120101a0330303022033030302a03362e30320556464f4e453a044d6f6f6e42224a313036475f56666f6e655f4d6f6f6e5f4231355f563030315f32303139313031384a2434376265346566382d623066622d346438342d396463642d6164393739613265613731663a0547657474794d21f4c91c60006801"
	UAData, err := hex.DecodeString(UAHex)
	if err != nil {
		t.Fatal(err)
	}

	routingInfo := []byte{0x08, 0x08, 0x08, 0x02}
	accountInfo := app.EmptyAccountInfo()
	accountInfo.SetRoutingInfo(routingInfo)

	_ = accountInfo.SetCliPayloadData(UAData)
	//accountInfo.SetStaticHdBase64Keys("sA97ZOg5pdWVucEi3oS4dS3JQAgCziUz0oN/NFAcyVI=","dPIvsCT1fOhO3mY0hgBPHHLVRaLOmzodB0oIRDUO9w4=")
	accountInfo.SetUserName(uint64(14233767293))
	_ = accountInfo.SetStaticHdKeys(priKey[:32], priKey[32:])

	appCli := app.NewWaAppCli(accountInfo)

	//appCli.NetWork.SetNetWorkProxy("socks5://127.0.0.1:51837")
	loginResult := appCli.WALogin()

	t.Log(loginResult.GetResult())

	sigs := make(chan os.Signal, 1)
	//signal.Notify 注册这个给定的通道用于接收特定信号。
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
