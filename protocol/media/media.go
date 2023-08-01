package media

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"
	"ws-go/libsignal/cipher"
	"ws-go/protocol/crypto/cbc"
	"ws-go/protocol/crypto/hkdf"
)

func getMediaKeys(mediaKey []byte, appInfo MediaType) (iv, cipherKey, macKey, refKey []byte, err error) {
	mediaKeyExpanded, err := hkdf.Expand(mediaKey, 112, string(appInfo))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return mediaKeyExpanded[:16], mediaKeyExpanded[16:48], mediaKeyExpanded[48:80], mediaKeyExpanded[80:], nil
}

func Download(proxy string, url string, mediaKey []byte, appInfo MediaType, fileLength int) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("no url present")
	}
	file, mac, err := downloadMedia(url, proxy)
	if err != nil {
		file, mac, err = downloadMedia(url, "")
		if err != nil {
			return nil, err
		}
	}
	iv, cipherKey, macKey, _, err := getMediaKeys(mediaKey, appInfo)
	if err != nil {
		return nil, err
	}
	if err = validateMedia(iv, file, macKey, mac); err != nil {
		return nil, err
	}
	data, err := cbc.Decrypt(cipherKey, iv, file)
	if err != nil {
		return nil, err
	}
	if fileLength != 0 {
		if len(data) != fileLength {
			return nil, fmt.Errorf("file length does not match. Expected: %v, got: %v", fileLength, len(data))
		}
	}

	return data, nil
}
func validateMedia(iv []byte, file []byte, macKey []byte, mac []byte) error {
	h := hmac.New(sha256.New, macKey)
	n, err := h.Write(append(iv, file...))
	if err != nil {
		return err
	}
	if n < 10 {
		return fmt.Errorf("hash to short")
	}
	if !hmac.Equal(h.Sum(nil)[:10], mac) {
		return fmt.Errorf("invalid media hmac")
	}
	return nil
}

func downloadMedia(urlReq, proxy string) (file []byte, mac []byte, err error) {
	client := &http.Client{}
	if proxy != "" {
		proxyStr := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxy)
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
		client = &http.Client{
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
		client = &http.Client{
			Transport: transport,
		}
	}
	resp, err := client.Get(urlReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("download failed with status code %d", resp.StatusCode)
	}
	if resp.ContentLength <= 10 {
		return nil, nil, fmt.Errorf("file to short")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	n := len(data)
	return data[:n-10], data[n-10 : n], nil
}

// 上传文件图
func UploadFor(proxy string, reader io.Reader, appInfo MediaType, hostname, auth string) (directPath, downloadURL string, mediaKey []byte, fileEncSha256 []byte, fileSha256 []byte, fileLength uint64, err error) {
	directPath, url, mediaKey, fileEncSha256, fileSha256, fileLength, errs := Upload(proxy, reader, appInfo, hostname, auth)
	//fmt.Println(directPath, url, mediaKey, fileSha256, fileEncSha256, fileLength)
	if errs != nil {
		return Upload("", reader, appInfo, hostname, auth)
	}
	return directPath, url, mediaKey, fileEncSha256, fileSha256, fileLength, errs
}

// Upload 上传
func Upload(proxy string, reader io.Reader, appInfo MediaType, hostname, auth string) (directPath, downloadURL string, mediaKey []byte, fileEncSha256 []byte, fileSha256 []byte, fileLength uint64, err error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", "", nil, nil, nil, 0, err
	}
	mediaKey = make([]byte, 32)
	rand.Read(mediaKey)
	iv, cipherKey, macKey, _, err := getMediaKeys(mediaKey, appInfo)
	if err != nil {
		return "", "", nil, nil, nil, 0, err
	}
	enc, err := cipher.EncryptS(cipherKey, iv, data)
	fileLength = uint64(len(data))
	h := hmac.New(sha256.New, macKey)
	h.Write(append(iv, enc...))
	mac := h.Sum(nil)[:10]
	sha := sha256.New()
	sha.Write(data)
	fileSha256 = sha.Sum(nil)
	sha.Reset()
	sha.Write(append(enc, mac...))
	fileEncSha256 = sha.Sum(nil)
	token := base64.URLEncoding.EncodeToString(fileEncSha256)
	q := url.Values{
		"auth":  []string{auth},
		"token": []string{token},
	}
	path := mediaTypeMap[appInfo]
	uploadURL := url.URL{
		Scheme:   "https",
		Host:     hostname,
		Path:     fmt.Sprintf("%s/%s", path, token),
		RawQuery: q.Encode(),
	}
	//https://{}/mms/{}/{}?auth={}&token={}
	//fmt.Println(uploadURL.String())
	body := bytes.NewReader(append(enc, mac...))
	req, err := http.NewRequest(http.MethodPost, uploadURL.String(), body)
	if err != nil {
		fmt.Println("--->", err.Error())
		return "", "", nil, nil, nil, 0, err
	}
	/*req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15")
	req.Header.Set("Origin", "https://web.whatsapp.com")
	req.Header.Set("Referer", "https://web.whatsapp.com/")*/
	client := &http.Client{}
	// Submit the request
	if proxy != "" {
		proxyStr := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxy)
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
		client = &http.Client{
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
		client = &http.Client{
			Transport: transport,
		}
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", "", nil, nil, nil, 0, fmt.Errorf("转换失败%d", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return "", "", nil, nil, nil, 0, fmt.Errorf("upload failed with status code %d", res.StatusCode)
	}
	var jsonRes map[string]string
	if err := json.NewDecoder(res.Body).Decode(&jsonRes); err != nil {
		return "", "", nil, nil, nil, 0, fmt.Errorf("转换失败%d", err.Error())
	}
	return jsonRes["direct_path"], jsonRes["url"], mediaKey, fileEncSha256, fileSha256, fileLength, nil
}

var mediaTypeMap = map[MediaType]string{
	MediaImage:    "/mms/image",
	MediaVideo:    "/mms/video",
	MediaDocument: "/mms/document",
	MediaAudio:    "/mms/audio",
}
