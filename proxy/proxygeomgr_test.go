package proxy

import (
	"testing"
	"time"
)

func TestNewProxyGeoMgr(t *testing.T) {
	pgm := NewProxyGeoMgr("cities.json", "", 0, nil)
	t.Logf("%#v\n", pgm)
}

func TestFetchProxy(t *testing.T) {
	//pgm := NewProxyGeoMgr("cities.json", "10.1.192.18:6379", 0, []string{"proxy_stable"})
	//t.Logf("%#v\n", pgm)
	NewProxyGeoMgr("cities.json", "10.1.192.18:6379", 0, []string{"proxy_stable"})
	time.Sleep(2 * time.Second)
}

func TestProxyLevel(t *testing.T) {
	pl := NewProxyLevel()

	p := NewProxy("http://122.11.37.52:8909")
	priv := &ProxyPrivate{LEVEL_NEW, 12}
	p.SetPrivate(priv)
	pl.Set(p, false)

	proxy := pl.Get()
	t.Logf("%#v\n", proxy.GetPrivate())
}
