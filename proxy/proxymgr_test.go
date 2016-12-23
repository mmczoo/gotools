package proxy

import "testing"

func TestInet(t *testing.T) {
	a := "100.12.1.1"
	t.Logf("%X\n", inetatoi(a))
	a = "1978.1.1.1"
	t.Logf("%x\n", inetatoi(a))
}

func TestMgr(t *testing.T) {
	pm := NewProxyMgr()

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

	pm.FeedBack(px)
	pm.FeedBack(px2)
	pm.FeedBack(px3)

	px = pm.Get()
	t.Logf("%#v %#v\n", px, px.GetPrivate())

}
