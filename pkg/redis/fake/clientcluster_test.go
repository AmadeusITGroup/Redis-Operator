package fake

import (
	"testing"

	"github.com/mediocregopher/radix.v2/redis"
)

func TestCmd(t *testing.T) {
	mock := NewClientCluster()
	mock.Resps["CLUSTER NODES"] = redis.NewRespSimple("someanswer")
	resp := mock.Cmd("CLUSTER", "NODES")
	val, err := resp.Str()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val != "someanswer" {
		t.Errorf("Expected '%s', got '%s'", "someanswer", val)
	}
}
