package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/redis.v2"

	ac "github.com/gansidui/ahocorasick"
	"github.com/mmczoo/goqueue"
	"github.com/xlvector/dlog"
)

/*
proxy level:
1 -- proxy available often
2 -- proxy used
3 -- proxy new

level change:
3 --> 2 --> 1
*/

const (
	LEVEL_ONE_SIZE   = 256
	LEVEL_TWO_SIZE   = 256
	LEVEL_THREE_SIZE = 256

	FETCH_PROXY_NUM  = 2
	FETCH_PROXY_INTV = 30 //second

	LEVEL_ONE   = 1
	LEVEL_TWO   = 2
	LEVEL_THREE = 3
	LEVEL_NEW   = 0
)

type ProxyPrivate struct {
	Level int
	Cid   int //city id
}

type ProxyLevel struct {
	l1 *goqueue.Queue
	l2 *goqueue.Queue
	l3 *goqueue.Queue
}

func NewProxyLevel() *ProxyLevel {
	return &ProxyLevel{
		l1: goqueue.New(LEVEL_ONE_SIZE),
		l2: goqueue.New(LEVEL_TWO_SIZE),
		l3: goqueue.New(LEVEL_THREE_SIZE),
	}
}

//get a proxy
func (p *ProxyLevel) Get() *Proxy {
	fmt.Println(p.l3.Size())
	pi, err := p.l1.GetNoWait()
	if err == nil {
		proxy, ok := pi.(*Proxy)
		if ok {
			return proxy
		}
	}
	pi, err = p.l2.GetNoWait()
	if err == nil {
		proxy, ok := pi.(*Proxy)
		if ok {
			return proxy
		}
	}
	pi, err = p.l3.GetNoWait()
	if err == nil {
		proxy, ok := pi.(*Proxy)
		if ok {
			return proxy
		}
	}

	return nil
}

//add a proxy, maybe be new proxy, used proxy
//see: proxy level
func (p *ProxyLevel) Set(proxy *Proxy, succ bool) {
	fmt.Println(p.l3.Size())
	priv, ok := proxy.GetPrivate().(*ProxyPrivate)
	if !ok {
		dlog.Info("proxy no private! %s", proxy.IP)
		return
	}

	var err error
	level := LEVEL_NEW
	if succ {
		switch priv.Level {
		case LEVEL_ONE:
			err = p.l1.PutNoWait(proxy)
			level = LEVEL_ONE
		case LEVEL_TWO:
			level = LEVEL_ONE
			err = p.l1.PutNoWait(proxy)
		case LEVEL_THREE:
			level = LEVEL_TWO
			err = p.l2.PutNoWait(proxy)
		//don't exist normally
		case LEVEL_NEW:
			level = LEVEL_THREE
			err = p.l3.PutNoWait(proxy)
		}
	} else {
		switch priv.Level {
		case LEVEL_ONE:
			level = LEVEL_TWO
			err = p.l2.PutNoWait(proxy)
		case LEVEL_TWO:
			level = LEVEL_THREE
			err = p.l3.PutNoWait(proxy)
		case LEVEL_THREE:
			//no operation
			//err = p.l3.PutNoWait(proxy)
		case LEVEL_NEW:
			level = LEVEL_THREE
			err = p.l3.PutNoWait(proxy)
		}
	}
	if err == nil {
		priv.Level = level
	}
	dlog.Info("setproxy: %v", err)
}

type ProxyGeoMgr struct {
	proxies     map[int]*ProxyLevel
	cities      map[string]int //
	redisclient *redis.Client
	pkeys       []string //redis keys

	ipgeo    *IpGeo
	cityAC   *ac.Matcher
	cityList []string
}

func (p *ProxyGeoMgr) initCities(cfile string) error {
	f, err := os.Open(cfile)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &p.cities)
	if err != nil {
		return err
	}

	return nil
}

//must be after initCities
func (p *ProxyGeoMgr) initProxyLevel() error {
	for _, cid := range p.cities {
		p.proxies[cid] = NewProxyLevel()
	}
	return nil
}

func (p *ProxyGeoMgr) initProxyFetch() {
	for _, key := range p.pkeys {
		err := p.fectchProxies(key, FETCH_PROXY_NUM)
		if err != nil {
			dlog.Warn("fetch fail! %s %s", key, err)
		}
	}

	go func() {
		t := time.NewTicker(FETCH_PROXY_INTV * time.Second)
		for {
			for _, key := range p.pkeys {
				<-t.C
				err := p.fectchProxies(key, FETCH_PROXY_NUM)
				if err != nil {
					dlog.Warn("fetch fail! %s %s", key, err)
				}
			}
		}
	}()
}

func NewProxyGeoMgr(cfile string, raddr string, rdb int, pkeys []string) *ProxyGeoMgr {
	if len(cfile) == 0 {
		dlog.Warn("no cities conf file!")
		return nil
	}

	pgm := &ProxyGeoMgr{
		proxies: make(map[int]*ProxyLevel),
		cities:  make(map[string]int),
		redisclient: redis.NewTCPClient(&redis.Options{
			Addr:        raddr,
			DB:          int64(rdb),
			DialTimeout: time.Duration(300) * time.Second,
		}),
		ipgeo: NewIpGeo(),
	}
	pgm.pkeys = pkeys

	err := pgm.initCities(cfile)
	if err != nil {
		dlog.Error("init city fail! %s", err)
		return nil
	}

	err = pgm.initProxyLevel()
	if err != nil {
		dlog.Error("init proxy level fail! %s", err)
		return nil
	}

	pgm.cityList = make([]string, 0, 512)
	for k, _ := range pgm.cities {
		pgm.cityList = append(pgm.cityList, k)
	}
	pgm.cityAC = ac.NewMatcher()
	pgm.cityAC.Build(pgm.cityList)
	dlog.Println("city dic: ", len(pgm.cityList))

	pgm.initProxyFetch()

	return pgm
}

func (p *ProxyGeoMgr) GetProxy(city string) *Proxy {
	cid, ok := p.cities[city]
	if !ok {
		return nil
	}

	pl, ok := p.proxies[cid]
	if !ok {
		return nil
	}

	return pl.Get()
}

func (p *ProxyGeoMgr) FindCityIdByName(name string) (int, error) {
	fmt.Println(name)
	ret := p.cityAC.Match(name)
	if len(ret) <= 0 {
		return 0, errors.New("not found")
	}

	cid, ok := p.cities[p.cityList[ret[0]]]
	if !ok {
		return 0, errors.New("not found")
	}
	return cid, nil

}

func (p *ProxyGeoMgr) Set(proxy *Proxy, succ bool) {
	priv, ok := proxy.GetPrivate().(*ProxyPrivate)
	if ok {
		pl, ok := p.proxies[priv.Cid]
		if ok {
			pl.Set(proxy, succ)
			return
		}
	}

	res := p.ipgeo.Query(strings.Split(proxy.IP, ":")[0])
	tmp := res.City + res.Prov + res.Addr
	cid, err := p.FindCityIdByName(tmp)
	if err != nil {
		dlog.Info("not found city: %s", proxy.IP)
		return
	}

	priv = &ProxyPrivate{
		Cid:   cid,
		Level: LEVEL_NEW,
	}
	proxy.SetPrivate(priv)
	pl, ok := p.proxies[priv.Cid]
	if ok {
		pl.Set(proxy, succ)
		return
	}
}

func (p *ProxyGeoMgr) fectchProxies(key string, length int64) error {
	if length <= 0 {
		return nil
	}
	cmd := p.redisclient.LRange(key, 0, length-1)
	if cmd == nil {
		return errors.New("fetch proxy fail!")
	}

	res, err := cmd.Result()
	if err != nil {
		return err
	}

	cnt := 0
	for _, addr := range res {
		proxy := NewProxy(addr)
		if proxy == nil {
			continue
		}

		cnt = cnt + 1
		go p.Set(proxy, false)
	}
	dlog.Info("fetch: %s %d", key, cnt)

	return nil
}
