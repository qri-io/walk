package lib

import (
	"testing"
)

func TestCrawl(t *testing.T) {
	stop := make(chan bool)
	crawl := NewCrawl()

	err := crawl.Start(stop)
	if err != nil {
		t.Error(err)
		return
	}
}
