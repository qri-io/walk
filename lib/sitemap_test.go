package lib

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestSitemapGeneratorType(t *testing.T) {
	expect := "SITEMAP"
	got := NewSitemapGenerator("", "", nil).Type()
	if expect != got {
		t.Errorf("type string mismatch. expected: %s, got: %s", expect, got)
	}
}

func TestSitemapGenerator(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "TestSitemapGenerator")
	if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	bcfg := NewBadgerConfig()
	bcfg.Dir = filepath.Join(tmp, "badger")
	bcfg.ValueDir = filepath.Join(tmp, "badger")
	conn, err := bcfg.DB()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmp, "map.json")
	sg := NewSitemapGenerator("test", path, conn)

	// bad url should fail on private "key" method
	sg.HandleResource(&Resource{URL: ":::::"})

	sg.HandleResource(exampleResourceA())
	sg.HandleResource(exampleResourceAa())

	if err := sg.FinalizeResources(); err != nil {
		t.Error(err)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}

	expect := []byte(`{
  "http://a.com": {
    "url": "https://www.a.com",
    "title": "",
    "timestamp": "1999-11-30T00:00:00Z",
    "status": 200,
    "redirects": null,
    "resources": null,
    "links": [
      "https://www.a.com/a",
      "https://www.a.com/b"
    ]
  },
  "http://a.com/a": {
    "url": "https://www.a.com/a",
    "title": "",
    "timestamp": "1999-11-30T00:00:00Z",
    "status": 200,
    "redirects": null,
    "resources": null,
    "links": [
      "https://www.a.com"
    ]
  }
}`)

	if !bytes.Equal(expect, data) {
		t.Errorf("generated sitemap mismatch. got:\n%s", string(data))
	}
}
