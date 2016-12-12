package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mmczoo/gotools/download"
	"github.com/patrickmn/go-cache"
	"github.com/xlvector/dlog"
)

type IpInfo struct {
	Ip       string
	Nation   string
	Prov     string
	City     string
	Addr     string
	Isp      string
	UpdateAt time.Time `ignore:"true"`
	//CreateAt time.Time

	Source int
}

type IpGeo struct {
	downloader *download.Downloader

	backch chan *IpInfo
	cache  *cache.Cache
}

const (
	BACK_PEOC_CHAN_LEN = 1024
	CACHE_TIMEOUT      = 24 * time.Hour
	WAIT_RES_TIMEOUT   = 2 * time.Second
)

func NewIpGeo() *IpGeo {
	ig := &IpGeo{}

	ig.downloader = download.NewDownloader(3)
	ig.backch = make(chan *IpInfo, BACK_PEOC_CHAN_LEN)
	ig.cache = cache.New(CACHE_TIMEOUT, CACHE_TIMEOUT/2)

	go ig.procCache()
	return ig
}

const (
	WSNO_PCONLINE = 1
	WSNO_OPENGPS  = 2
	WSNO_521PHP   = 3
	WSNO_SINA     = 4
)

var websiteno = []int{WSNO_PCONLINE, WSNO_OPENGPS, WSNO_521PHP, WSNO_SINA}

func (p *IpGeo) procCache() {
	for v := range p.backch {
		if v == nil {
			continue
		}
		p.cache.Set(v.Ip, v, cache.DefaultExpiration)
	}
}

/*
returl nil if query fail
*/
func (p *IpGeo) Query(ip string) *IpInfo {
	r, ok := p.cache.Get(ip)
	if ok {
		v, ok := r.(*IpInfo)
		if ok {
			return v
		}
	}

	ch := make(chan *IpInfo, len(websiteno))
	chflag := true
	defer func() {
		chflag = false
		close(ch)
	}()

	for _, v := range websiteno {
		go func(swno int) {
			r := p.queryFromInternet(ip, swno)
			if r != nil {
				if chflag {
					ch <- r
				}
				p.backch <- r
			}
		}(v)
	}

	var res *IpInfo
	//for {
	select {
	case <-time.After(WAIT_RES_TIMEOUT):
		dlog.Info("timeout: %s", ip)
		res = nil
	case res = <-ch:
	}
	//}

	//res := p.queryFromInternet(ip, WSNO_PCONLINE)
	//res := p.queryFromInternet(ip, WSNO_OPENGPS)
	//res := p.queryFromInternet(ip, WSNO_521PHP)
	//res := p.queryFromInternet(ip, WSNO_SINA)

	return res
}

func wsnoToUrl(ip string, wsno int) string {
	url := ""
	switch wsno {
	case WSNO_PCONLINE:
		url = fmt.Sprintf("http://whois.pconline.com.cn/ipJson.jsp?callback=testJson&ip=%s", ip)
	case WSNO_OPENGPS:
		url = fmt.Sprintf("https://www.opengps.cn/Data/IP/IPLocHiAcc.ashx?ip=%s", ip)
	case WSNO_521PHP:
		url = fmt.Sprintf("http://www.521php.com/api/ip.php?ip=%s&format=json&charset=utf8", ip)
	case WSNO_SINA:
		url = fmt.Sprintf("http://int.dpool.sina.com.cn/iplookup/iplookup.php?format=js&ip=%s", ip)
	}
	return url
}

var headers = map[string]string{
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	"Accept-Encoding": "gzip, deflate",
	"Accept-Language": "zh-CN,zh;q=0.8",
	"User-Agent":      "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.116 Safari/537.36",
}

func (p *IpGeo) queryFromInternet(ip string, wsno int) *IpInfo {
	url := wsnoToUrl(ip, wsno)
	if len(url) == 0 {
		return nil
	}

	dlog.Info("url:%s", url)
	data, err := p.downloader.Get(url, headers)
	if err != nil {
		dlog.Warn("get: fail! %d", wsno)
		return nil
	}

	res := extract(data, wsno)
	if res != nil {
		res.Ip = ip
		res.Source = wsno
		//res.CreateAt = time.Now()
		//res.UpdateAt = time.Now()
	}
	return res
}

type OpengpsResVal struct {
	Ip   string `json:"ip"`
	Addr string `json:"address"`
}
type OpengpsRes struct {
	Sccuess bool             `json:"success"`
	Values  []*OpengpsResVal `json:"values"`
}
type PconlineRes struct {
	Ip   string `json:"ip"`
	Prov string `json:"prov"`
	City string `json:"city"`
	Addr string `json:"addr"`
	Err  string `json:"err"`
}
type PHP521Res struct {
	Ip    string `json:"ip"`
	IpLoc string `json:"iplocation"`
}
type SinaRes struct {
	Ip   string `json:"ip"`
	Prov string `json:"province"`
	City string `json:"city"`
	Addr string `json:"district"`
	Isp  string `json:"isp"`
}

func extract(data []byte, wsno int) *IpInfo {
	if len(data) < 20 {
		dlog.Warn("extract: fail! %d", wsno)
		return nil
	}

	ipinfo := &IpInfo{}
	switch wsno {
	case WSNO_PCONLINE:
		pos := bytes.Index(data, []byte("testJson("))
		epos := bytes.LastIndex(data, []byte(");"))
		if pos <= 0 || epos <= 0 {
			dlog.Warn("extract: index! %d", wsno)
			return nil
		}
		result := &PconlineRes{}
		err := json.Unmarshal(data[pos+9:epos], result)
		//if err != nil || len(result.Err) != 0 {
		if err != nil {
			dlog.Warn("extract: jsonfail! %d %v %s", wsno, err, result.Err)
			fmt.Println(string(data[pos+9 : epos]))
			return nil
		}
		ipinfo.Addr = result.Addr
		ipinfo.Prov = result.Prov
		ipinfo.City = result.City
		ipinfo.Addr = result.Addr
	case WSNO_OPENGPS:
		result := &OpengpsRes{}
		err := json.Unmarshal(data, result)
		if err != nil {
			dlog.Warn("extract: jsonfail! %d", wsno)
			return nil
		}
		if !result.Sccuess || len(result.Values) <= 0 {
			dlog.Warn("extract: query fail! %d", wsno)
			return nil

		}
		ipinfo.Addr = result.Values[0].Addr
	case WSNO_521PHP:
		pos := bytes.Index(data, []byte("({"))
		epos := bytes.LastIndex(data, []byte("})"))
		if pos < 0 || epos <= 0 {
			dlog.Warn("extract: index! %d", wsno)
			fmt.Println(string(data))
			return nil
		}
		result := &PHP521Res{}
		err := json.Unmarshal(data[pos+1:epos+1], result)
		if err != nil {
			dlog.Warn("extract: jsonfail! %d", wsno)
			return nil
		}
		ipinfo.Addr = result.IpLoc
	case WSNO_SINA:
		pos := bytes.Index(data, []byte("{"))
		epos := bytes.LastIndex(data, []byte("};"))
		if pos <= 0 || epos <= 0 {
			dlog.Warn("extract: index! %d", wsno)
			return nil
		}
		result := &SinaRes{}
		err := json.Unmarshal(data[pos:epos+1], result)
		if err != nil {
			dlog.Warn("extract: jsonfail! %d", wsno)
			return nil
		}
		ipinfo.Addr = result.Addr
		ipinfo.Prov = result.Prov
		ipinfo.City = result.City
		ipinfo.Isp = result.Isp
	}

	if len(ipinfo.Addr) == 0 &&
		len(ipinfo.City) == 0 &&
		len(ipinfo.Prov) == 0 {
		return nil
	}
	return ipinfo
}
