package proxy

import "testing"

func TestQuery(t *testing.T) {
	p := NewIpGeo()

	res := p.Query("218.6.160.93")
	t.Logf("---%#v\n", res)
	res = p.Query("218.6.160.93")
	t.Logf("---%#v\n", res)
	/*
		res = p.Query("66.102.251.33")
		t.Logf("%#v\n", res)
		res = p.Query("115.159.231.139")
		t.Logf("%#v\n", res)
		res = p.Query("218.30.108.232")
		t.Logf("%#v\n", res)
		res = p.Query("111.206.172.150")
		t.Logf("%#v\n", res)
		res = p.Query("111.206.172.150")
		res = p.Query("115.159.231.139")
		t.Logf("%#v\n", res)
		res = p.Query("115.159.231.139")
	*/
}
