package proxy

import (
	"strconv"
	"strings"
	"time"

	"github.com/mmczoo/goqueue"
	"github.com/xlvector/dlog"
)

type ProxyMgr struct {
	l1 *goqueue.Queue
	l2 *goqueue.Queue
	l3 *goqueue.Queue

	ipst   map[uint32]uint
	fbipst map[uint32]uint
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

func NewProxyMgr() *ProxyMgr {
	return &ProxyMgr{
		l1: goqueue.New(Q_MAX_SIZE),
		l2: goqueue.New(Q_MAX_SIZE),
		l3: goqueue.New(Q_MAX_SIZE),

		ipst:   make(map[uint32]uint),
		fbipst: make(map[uint32]uint),
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

	val, ok := p.ipst[private.Ipv4]
	if ok {
		p.ipst[private.Ipv4] = val + 1
	} else {
		p.ipst[private.Ipv4] = 1
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
