package proxy

import "testing"

func TestInet(t *testing.T) {
	a := "100.12.1.1"
	t.Logf("%X\n", inetatoi(a))
	a = "1978.1.1.1"
	t.Logf("%x\n", inetatoi(a))
}

func TestMgr(t *testing.T) {
	pm := NewProxyMgr(&Redis{"10.1.192.18:6379", 1, 300})

	addr := "http://1.1.1.1:90"
	pm.Add(addr)
	addr = "http://hhhhd:99999@1.4.1.1:90"
	pm.Add(addr)

	px := pm.Get()
	//fmt.Printf("=======%#v\n", px)
	t.Logf("=======%#v\n", px)

	px2 := pm.Get()
	t.Logf("%#v\n", px2)

	px3 := pm.Get()
	t.Logf("%#v\n", px3)

	t.Logf("%#v\n", pm.GetIpst())

	pm.FeedBack(px)
	pm.FeedBack(px2)
	pm.FeedBack(px3)

	t.Logf("%#v\n", pm.GetFBIpst())

	px = pm.Get()
	t.Logf("%#v\n", pm.GetIpst())
	t.Logf("%#v %#v\n", px, px.GetPrivate())

}

func TestInetI(t *testing.T) {
	t.Logf("%v\n", inetitoa(0x02030902))
	t.Logf("==%v\n", inetitoa(0x05010601))
}

func TestMgr2(t *testing.T) {
	pm := NewProxyMgr(&Redis{"10.1.192.18:6379", 0, 300})

	addr := "http://1.1.1.1:90"
	pm.Add(addr)
	addr = "http://hhhhd:99999@1.4.1.1:90"
	pm.Add(addr)

	px := pm.Get()
	//fmt.Printf("=======%#v\n", px)
	t.Logf("=======%#v\n", px)

	px2 := pm.Get()
	t.Logf("%#v\n", px2)

	px3 := pm.Get()
	t.Logf("%#v\n", px3)

	t.Logf("%#v\n", pm.GetIpst())

	pm.FeedBack(px)
	pm.FeedBack(px2)
	pm.FeedBack(px3)

	t.Logf("%#v\n", pm.GetFBIpst())

	px = pm.Get()
	t.Logf("%#v\n", pm.GetIpst())
	t.Logf("%#v %#v\n", px, px.GetPrivate())

}
