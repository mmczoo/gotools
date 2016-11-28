package download

import "testing"

func TestGet(t *testing.T) {

	d := NewDownloader()

	data, err := d.Get("http://studygolang.com/articles/1245", nil)
	//data, err := d.Get("http://git.guazi-corp.com/chendansi/crawlWebTool/blob/master/crawlServer.go", nil)
	if err != nil {
		t.Error(err)
	}

	//fmt.Println("===========" + string(data))
	t.Log("====" + string(data))
}
