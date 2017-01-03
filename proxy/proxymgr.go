package proxy

import (
	"log"
	"strconv"
	"strings"
	"time"

	"gopkg.in/redis.v2"

	"github.com/mmczoo/goqueue"
	"github.com/seefan/gossdb"
	"github.com/xlvector/dlog"
)

type Redis struct {
	Host    string `json:"host"`
	DB      int64  `json:"db"`
	Timeout int64  `json:"timeout"`

	Keys map[string]int64 `json:"keys"`

	RefreshIntv int `json:"refreshintv"`
}

type Ssdb struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	Keys map[string]int64 `json:"keys"`

	RefreshIntv int `json:"refreshintv"`
}

type ProxyMgr struct {
	l1 *goqueue.Queue
	l2 *goqueue.Queue
	l3 *goqueue.Queue

	ipst   map[uint32]uint
	fbipst map[uint32]uint

	redisclient *redis.Client
	rediscfg    *Redis

	ssdbcfg    *Ssdb
	pool       *gossdb.Connectors
	ssdbclient *gossdb.Client
}

const (
	P_LEVEL_ONE   = 0
	P_LEVEL_TWO   = 1
	P_LEVEL_THREE = 2

	Q_MAX_SIZE = 2000
)

type PrivateData struct {
	Ipv4     uint32
	Level    int
	LastTime int64
}

func NewProxyMgrWithSsdb(r *Ssdb) *ProxyMgr {
	if r == nil && len(r.Keys) <= 0 {
		return nil
	}
	pm := &ProxyMgr{
		l1: goqueue.New(Q_MAX_SIZE),
		l2: goqueue.New(Q_MAX_SIZE),
		l3: goqueue.New(Q_MAX_SIZE),

		ipst:   make(map[uint32]uint),
		fbipst: make(map[uint32]uint),

		ssdbcfg: r,
	}

	pool, err := gossdb.NewPool(&gossdb.Config{
		Host:             r.Host,
		Port:             r.Port,
		MinPoolSize:      5,
		MaxPoolSize:      50,
		AcquireIncrement: 1,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	c, err := pool.NewClient()
	if err != nil {
		log.Fatal(err)
		return nil
	}
	pm.ssdbclient = c
	pm.pool = pool

	if pm.ssdbcfg.RefreshIntv < 5 {
		pm.ssdbcfg.RefreshIntv = 5
	}

	go pm.refreshFromSsdb()

	return pm
}

func NewProxyMgr(r *Redis) *ProxyMgr {
	if r == nil && len(r.Keys) <= 0 {
		return nil
	}
	pm := &ProxyMgr{
		l1: goqueue.New(Q_MAX_SIZE),
		l2: goqueue.New(Q_MAX_SIZE),
		l3: goqueue.New(Q_MAX_SIZE),

		ipst:   make(map[uint32]uint),
		fbipst: make(map[uint32]uint),

		rediscfg: r,
		redisclient: redis.NewTCPClient(&redis.Options{
			Addr:        r.Host,
			DB:          r.DB,
			DialTimeout: time.Duration(r.Timeout) * time.Second,
		}),
	}

	if pm.rediscfg.RefreshIntv < 5 {
		pm.rediscfg.RefreshIntv = 5
	}

	go pm.refresh()

	return pm
}

func (p *ProxyMgr) refreshFromSsdb() {
	try := 0
	t := time.NewTicker(time.Duration(p.ssdbcfg.RefreshIntv) * time.Second)

	for {
		for k, v := range p.ssdbcfg.Keys {
			ret, err := p.ssdbclient.Zrrange(k, 0, v)
			if err != nil {
				try++
				if try >= 3 {
					<-t.C
					try = 0
				}
				continue
			}
			dlog.Info("zrrange: %d", len(ret))
			for ip, _ := range ret {
				p.Add(ip)
			}
			<-t.C
		}
	}
}

func (p *ProxyMgr) refresh() {
	t := time.NewTicker(time.Duration(p.rediscfg.RefreshIntv) * time.Second)

	for {
		for k, v := range p.rediscfg.Keys {
			slc := p.redisclient.LRange(k, 0, v)
			ips := slc.Val()
			dlog.Info("lrange: %d", len(ips))
			for _, ip := range ips {
				p.Add(ip)
			}

			<-t.C
		}
	}
}

func NewPrivateData(ipv4 uint32) *PrivateData {
	return &PrivateData{
		Ipv4:  ipv4,
		Level: P_LEVEL_THREE,
	}
}

//0: error
func inetatoi(ip string) uint32 {
	tmp := strings.Split(ip, ".")
	if len(tmp) != 4 {
		return 0
	}
	a1, err := strconv.Atoi(tmp[0])
	if err != nil || a1 > 255 {
		return 0
	}
	a2, err := strconv.Atoi(tmp[1])
	if err != nil || a2 > 255 {
		return 0
	}
	a3, err := strconv.Atoi(tmp[2])
	if err != nil || a3 > 255 {
		return 0
	}
	a4, err := strconv.Atoi(tmp[3])
	if err != nil || a4 > 255 {
		return 0
	}

	return uint32(a4*16777216 + a3*65536 + a2*256 + a1)
}

//addr: http://1.1.1.1:80
func (p *ProxyMgr) Add(addr string) {
	px := NewProxy(addr)
	if px == nil {
		dlog.Warn("proxymgr add fail: %s", addr)
		return
	}
	tmp := strings.Split(px.IP, ":")
	intip := inetatoi(tmp[0])
	if intip == 0 {
		//fmt.Printf("%#v\n", px)
		dlog.Warn("proxymgr inetatoi fail: %s", addr)
		return
	}

	px.SetPrivate(NewPrivateData(intip))

	p.l3.PutNoWait(px)
}

func (p *ProxyMgr) FeedBack(px *Proxy) {
	if px == nil {
		return
	}

	priv := px.GetPrivate()
	private, ok := priv.(*PrivateData)
	if !ok {
		return
	}

	val, ok := p.fbipst[private.Ipv4]
	if ok {
		p.fbipst[private.Ipv4] = val + 1
	} else {
		p.fbipst[private.Ipv4] = 1
	}

	switch private.Level {
	case P_LEVEL_ONE:
		p.l1.PutNoWait(px)
	case P_LEVEL_TWO:
		private.Level = P_LEVEL_ONE
		p.l1.PutNoWait(px)
	case P_LEVEL_THREE:
		private.Level = P_LEVEL_TWO
		p.l2.PutNoWait(px)
	}
}

func (p *ProxyMgr) Get() *Proxy {
	var rpx *Proxy = nil
	var ok bool
	px, err := p.l1.GetNoWait()
	if err == nil {
		rpx, ok = px.(*Proxy)
		if ok {
			goto lab_succ
		}
	}
	px, err = p.l2.GetNoWait()
	if err == nil {
		rpx, ok = px.(*Proxy)
		if ok {
			goto lab_succ
		}
	}
	px, err = p.l3.GetNoWait()
	if err == nil {
		rpx, ok = px.(*Proxy)
		if ok {
			goto lab_succ
		}
	}
	return nil

lab_succ:
	priv := rpx.GetPrivate()
	private, ok := priv.(*PrivateData)
	if ok {
		val, ok := p.ipst[private.Ipv4]
		if ok {
			p.ipst[private.Ipv4] = val + 1
		} else {
			p.ipst[private.Ipv4] = 1
		}

		private.LastTime = time.Now().Unix()
	}

	return rpx
}

func inetitoa(ip uint32) string {
	a1 := int(ip & 0x000000FF)
	a2 := int((ip >> 8) & 0x000000FF)
	a3 := int((ip >> 16) & 0x000000FF)
	a4 := int((ip >> 24) & 0x000000FF)

	return strconv.Itoa(a1) + "." + strconv.Itoa(a2) + "." + strconv.Itoa(a3) + "." + strconv.Itoa(a4)
}

func (p *ProxyMgr) GetFBIpst() map[string]uint {
	max := 1000
	ret := make(map[string]uint)

	for k, v := range p.fbipst {
		max -= 1
		if max <= 0 {
			break
		}
		ip := inetitoa(k)
		ret[ip] = v
	}

	return ret
}

func (p *ProxyMgr) GetIpst() map[string]uint {
	max := 1000
	ret := make(map[string]uint)

	for k, v := range p.ipst {
		max -= 1
		if max <= 0 {
			break
		}
		ip := inetitoa(k)
		ret[ip] = v
	}
	return ret
}
