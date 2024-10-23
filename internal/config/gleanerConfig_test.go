package config

import (
	"testing"
)

func TestGleanerConfig(t *testing.T) {
	v, err := ReadGleanerConfig("gleanerconfig.yaml", "../../test")
	if err != nil {
		t.Fatal(err)
	}
	res := v.Sub("minio")
	if res == nil {
		t.Fatal("no minio config")
	}
}
