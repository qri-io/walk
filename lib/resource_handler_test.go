package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewResourceHandlers(t *testing.T) {
	t.Skip("TODO")
}

func TestCBORResourceFileWriter(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "TestCBORResourceFileWriter")
	if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	rh, err := NewCBORResourceFileWriter(tmp)
	if err != nil {
		t.Fatal(err)
	}

	expectType := "CBOR"
	gotType := rh.Type()
	if expectType != gotType {
		t.Errorf("type mismatch. expected: %s, got: %s", expectType, gotType)
	}

	r := exampleResourceA()
	// should error b/c no hash
	rh.HandleResource(r)

	r.Hash = "not_actually_a_hash"
	rh.HandleResource(r)
}
