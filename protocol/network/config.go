package network

import (
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"time"
)

var (
	NetProxyEmptyError = errors.New("The network proxy string is empty")
)

// NoiseClientConfig
type NoiseClientConfig struct {
	// NetProxy 网络代理
	NetWorkProxy string
}

// SetNetWorkProxy
func (b *NoiseClientConfig) SetNetWorkProxy(s string) {
	if b.NetWorkProxy == "" || s != b.NetWorkProxy {
		b.NetWorkProxy = s
	}
}
func (c *NoiseClientConfig) GetNetWorkProxyStr() string {
	return c.NetWorkProxy
}

// GetNetProxy
func (c *NoiseClientConfig) GetNetWorkProxy() (proxy.Dialer, error) {
	if c.NetWorkProxy == "" {
		return nil, NetProxyEmptyError
	}
	// Parse Socket Url
	u, err := url.Parse(c.NetWorkProxy)
	if err != nil {
		return nil, err
	}
	dialer := proxy.FromEnvironment()
	if u.Scheme == "http" {
		pawd, _ := u.User.Password()
		auth := &proxy.Auth{
			User:     u.User.Username(),
			Password: pawd,
		}
		dialer, err = proxy.SOCKS5("tcp", u.Host, auth, &net.Dialer{Timeout: 20 * time.Second})
	} else {
		//socket5
		dialer, err = proxy.FromURL(u, proxy.Direct)
	}
	if err != nil {
		fmt.Println("代理出现异常error", err.Error())
		return nil, err
	}
	return dialer, nil
}
