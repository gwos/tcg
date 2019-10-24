package controller

import (
	"testing"
)

func TestStartServer (t *testing.T) {
	err := StartServer(false, 8081)
	if err != nil {
		t.Error(err)
		return
	}
}

