package api

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// CMD-07: network-frozen maps to 423 with its own problem code.
func TestNetworkFrozenMapsTo423(t *testing.T) {
	ae, ok := fromErr(domain.Errf(domain.CodeNetworkFrozen, "read-only"))
	if !ok || ae.status != 423 || ae.code != "network-frozen" {
		t.Fatalf("got %+v ok=%v", ae, ok)
	}
}
