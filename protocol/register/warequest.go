package register

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var CliPool = sync.Pool{New: func() interface{} {
	return &http.Client{}
}}

type WaRequestTask struct {
	HttpProxy  *url.URL
	Parameter  *url.Values
	Api        string
	Header     http.Header
	HttpMethod string
	// 任务选项 可实现在执行请求任务之前调用该函数
	OptionMethod func(task *WaRequestTask) error

	reqData []byte
}

func (r *WaRequestTask) setReqData(reqData []byte) {
	r.reqData = reqData
}

// Execute 执行
func (r *WaRequestTask) Execute() (result []byte, err error) {
	if r.OptionMethod != nil {
		_ = r.OptionMethod(r)
	}

	if len(r.reqData) == 0 && r.Parameter != nil {
		r.reqData = []byte(r.Parameter.Encode())
	} else if len(r.reqData) == 0 && r.Parameter == nil {
		// 执行任务失败缺少提交参数
		return nil, errors.New("Failed to execute task. Submit parameter is missing")
	}

	if r.HttpMethod == http.MethodGet {
		r.Api = r.Api + string(r.reqData)
		r.reqData = []byte{}
	}
	resp, err := httpRequest(r.HttpMethod, r.Api, r.reqData, r.Header, r.HttpProxy)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	return body, nil
}

func httpRequest(method, api string, data []byte, headers http.Header, proxy *url.URL) (*http.Response, error) {
	req, err := http.NewRequest(method, api, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header = headers

	cli := CliPool.Get().(*http.Client)
	defer CliPool.Put(cli)

	//
	if proxy.Host != "" {
		proxyStr := func(_ *http.Request) (*url.URL, error) {
			return proxy, nil
		}
		transport := &http.Transport{
			IdleConnTimeout:       4 * time.Second,
			TLSHandshakeTimeout:   4 * time.Second,
			ResponseHeaderTimeout: 4 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: proxyStr,
		}
		cli = &http.Client{
			Transport: transport,
		}
	} else {
		transport := &http.Transport{
			IdleConnTimeout:       4 * time.Second,
			TLSHandshakeTimeout:   4 * time.Second,
			ResponseHeaderTimeout: 4 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}, // 使用环境变量的代理
			Proxy: http.ProxyFromEnvironment,
		}
		cli = &http.Client{
			Transport: transport,
		}
	}
	return cli.Do(req)
}
