package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

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
