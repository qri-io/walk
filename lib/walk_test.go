package lib

import (
	"testing"
)

func TestNewWalk(t *testing.T) {
	tc := NewHTTPDirTestCase(t, "testdata/qri_io")
	s := tc.Server()

	walk, stop, err := NewWalk(tc.Config(s))
	if err != nil {
		t.Fatal(err.Error())
	}

	walk.Start(stop)
}
