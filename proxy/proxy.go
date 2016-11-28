package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

type Proxy struct {
	IP        string
	Type      string
	Username  string
	Password  string
	BlockTime time.Time
	LastTime  int64
}

type ProxyMgr struct {
	proxies []*Proxy
	pos     int
}

func (self *ProxyMgr) GetProxy() *Proxy {
	self.pos += 1
	if self.pos >= len(self.proxies) {
		self.pos = 0
		return nil
	}
	return self.proxies[self.pos]
}

func (self *ProxyMgr) InitProxy(fname string) {
	fi, err := os.Open(fname)
	if err != nil {
		fmt.Println("open proxy file fail!", err)
		return
	}
	defer fi.Close()
	data, err := ioutil.ReadAll(fi)
	if err != nil {
		fmt.Println("read proxy file fail!", err)
		return
	}

	var tmp []string
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		fmt.Println("unjson proxy file fail!", err)
		return
	}

	self.proxies = make([]*Proxy, len(tmp), len(tmp))
	pos := 0
	for _, p := range tmp {
		tp := NewProxy(p)
		if tp != nil {
			self.proxies[pos] = tp
			pos += 1
		}
	}
	fmt.Println("init proxy succ!", pos)
	return
}

func NewProxy(buf string) *Proxy {
	typeOthers := strings.SplitN(buf, "://", 2)
	if len(typeOthers) != 2 {
		return nil
	}
	authOthers := strings.SplitN(typeOthers[1], "@", 2)
	if len(authOthers) == 1 {
		return &Proxy{
			IP:        authOthers[0],
			Type:      typeOthers[0],
			Username:  "",
			Password:  "",
			BlockTime: time.Now(),
		}
	} else if len(authOthers) == 2 {
		userPwd := strings.SplitN(authOthers[0], ":", 2)
		return &Proxy{
			IP:        authOthers[1],
			Type:      typeOthers[0],
			Username:  userPwd[0],
			Password:  userPwd[1],
			BlockTime: time.Now(),
		}
	}
	return nil
}

func (p *Proxy) String() string {
	ret := p.Type + "://"
	if len(p.Username) > 0 {
		ret += p.Username + ":" + p.Password + "@"
	}
	ret += p.IP
	return ret
}

func (p *Proxy) Available() bool {
	if strings.Contains(p.IP, "127.0.0.1") {
		return true
	}
	conn, err := net.DialTimeout("tcp", p.IP, time.Second*5)
	if err != nil {
		//util.SlackMessage(config.Instance.SlackApi, "#crawler", "higgs", "proxy "+p.IP+" is not available")
		return false
	}
	conn.Close()
	return true
}

func (p *Proxy) IsBlock() bool {
	return p.BlockTime.Sub(time.Now()).Seconds() > 0.0
}

const (
	DEFAULT_TMPL = "default"
)
