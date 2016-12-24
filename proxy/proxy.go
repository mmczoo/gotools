package proxy

import (
	"net"
	"strings"
	"time"
)

type Proxy struct {
	IP        string
	Type      string
	Username  string
	Password  string
	BlockTime time.Time
	private   interface{}
}

func (p *Proxy) SetPrivate(private interface{}) {
	p.private = private
}

func (p *Proxy) GetPrivate() interface{} {
	return p.private
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
