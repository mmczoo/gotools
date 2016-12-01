package download

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	hproxy "github.com/mmczoo/caresm/src/phoneserver/proxy"

	"golang.org/x/net/proxy"
	"golang.org/x/net/publicsuffix"

	"github.com/axgle/mahonia"
	"github.com/xlvector/dlog"
)

type Downloader struct {
	Client *http.Client
	Page   []byte
}

func (self *Downloader) SetProxy(p *hproxy.Proxy) {
	transport := &http.Transport{
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: time.Second * 30,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MaxVersion:         tls.VersionTLS12,
			MinVersion:         tls.VersionTLS10,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}

	if p == nil {
		self.Client.Transport = transport
		return
	}

	if p.Type == "socks5" {
		var auth *proxy.Auth
		if len(p.Username) > 0 && len(p.Password) > 0 {
			auth = &proxy.Auth{
				User:     p.Username,
				Password: p.Password,
			}
		} else {
			auth = &proxy.Auth{}
		}
		forward := proxy.FromEnvironment()
		dialSocks5Proxy, err := proxy.SOCKS5("tcp", p.IP, auth, forward)
		if err != nil {
			dlog.Warn("SetSocks5 Error:%s", err.Error())
			return
		}
		transport.Dial = dialSocks5Proxy.Dial
	} else if p.Type == "http" || p.Type == "https" {
		transport.Dial = func(netw, addr string) (net.Conn, error) {
			timeout := time.Second * 30
			deadline := time.Now().Add(timeout)
			c, err := net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		}
		proxyUrl, err := url.Parse(p.String())
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyUrl)
		}
	} else if p.Type == "socks4" {
		surl := "socks4://" + p.IP
		rsurl, err := url.Parse(surl)
		if err != nil {
			dlog.Warn("socks4 url parse: %v", err)
			return
		}
		forward := proxy.FromEnvironment()
		dialersocks4, err := proxy.FromURL(rsurl, forward)
		if err != nil {
			dlog.Warn("SetSocks4 Error:%s", err.Error())
			return
		}
		transport.Dial = dialersocks4.Dial
	}

	self.Client.Transport = transport
	dlog.Warn("use proxy: %s", p.String())
}

const (
	DEFAULT_USERAGENT = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.80 Safari/537.36"
)

func NewDownloader(wait int) *Downloader {
	return NewDownloaderWithJar(wait, false)
}

func NewDownloaderWithJar(wait int, isjar bool) *Downloader {
	d := &Downloader{
		Client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				dlog.Warn("CheckRedirect URL:%s", req.URL.String())
				return nil
			},
			Timeout: time.Second * time.Duration(wait),
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					deadline := time.Now().Add(time.Duration(wait) * time.Second)
					c, err := net.DialTimeout(netw, addr, time.Second*time.Duration(wait))
					if err != nil {
						return nil, err
					}
					c.SetDeadline(deadline)
					return c, nil

				},
			},
		},
	}
	if isjar {
		options := cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		}
		jar, err := cookiejar.New(&options)
		if err != nil {
			dlog.Error("new cookiejar fail! %s", err)
		} else {
			d.Client.Jar = jar
		}
	}
	return d
}

func decodeCharset(body, contentTypeHeader string) (string, string) {
	tks := strings.Split(contentTypeHeader, ";")
	var content_type, charset string

	if len(tks) == 1 {
		content_type = strings.ToLower(tks[0])
	}
	if len(tks) == 2 {
		kv := strings.Split(tks[1], "=")
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "charset" {
			return strings.ToLower(tks[0]), strings.ToLower(kv[1])
		}
	}

	reg := regexp.MustCompile("meta[^<>]*[ ]{1}charset=\"([^\"]+)\"")
	result := reg.FindAllStringSubmatch(string(body), 1)
	if len(result) > 0 {
		group := result[0]
		if len(group) > 1 {
			charset = group[1]
		}
	}
	return content_type, charset
}

func (s *Downloader) constructPage(resp *http.Response) error {
	defer resp.Body.Close()
	body := make([]byte, 0)
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ := gzip.NewReader(resp.Body)
		defer reader.Close()
		for {
			buf := make([]byte, 1024)
			n, err := reader.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				break
			}
			body = append(body, buf[:n]...)
		}
	default:
		for {
			buf := make([]byte, 1024)
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				break
			}
			body = append(body, buf[:n]...)
		}
	}
	s.Page = body

	contentType, charset := decodeCharset(string(body), resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "image") && (strings.HasPrefix(charset, "gb") || strings.HasPrefix(charset, "GB")) {
		enc := mahonia.NewDecoder("gbk")
		cbody := []byte(enc.ConvertString(string(body)))
		s.Page = cbody
	}
	return nil
}

func (s *Downloader) Get(link string, header map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		dlog.Warn("new req error: %v", err)
		return nil, err
	}
	req.Header.Set("User-Agent", DEFAULT_USERAGENT)
	//req.Header.Set("Referer", s.LastPageUrl)
	if header != nil {
		for name, value := range header {
			req.Header.Set(name, value)
		}
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		dlog.Warn("do req error: %v", err)
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("nil resp")
	}
	err = s.constructPage(resp)
	if err != nil {
		return nil, err
	}
	return s.Page, nil
}

func (s *Downloader) Post(link string, params map[string]string, header map[string]string) ([]byte, error) {
	uparams := url.Values{}
	for k, v := range params {
		uparams.Set(k, v)
	}
	dlog.Info("post paramter:%v", uparams)
	req, err := http.NewRequest("POST", link, strings.NewReader(uparams.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", DEFAULT_USERAGENT)
	//req.Header.Set("Referer", s.LastPageUrl)
	if header != nil {
		for name, value := range header {
			req.Header.Set(name, value)
		}
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	err = s.constructPage(resp)
	if err != nil {
		return nil, err
	}
	return s.Page, nil
}

func (s *Downloader) PostRaw(link string, data []byte, header map[string]string) ([]byte, error) {
	req, err := http.NewRequest("POST", link, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain; charset=UTF-8")
	req.Header.Set("User-Agent", DEFAULT_USERAGENT)
	//req.Header.Set("Referer", s.LastPageUrl)
	if header != nil {
		for name, value := range header {
			req.Header.Set(name, value)
		}
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	err = s.constructPage(resp)
	if err != nil {
		return nil, err
	}
	return s.Page, nil
}
