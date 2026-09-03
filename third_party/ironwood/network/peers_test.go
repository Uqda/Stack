package network

import (
	"errors"
	"testing"

	"github.com/Arceliar/ironwood/types"
)

func TestCheckedPeerMessageSize(t *testing.T) {
	maxInt := uint64(^uint(0) >> 1)

	if got, err := checkedPeerMessageSize(1024, 1024); err != nil || got != 1024 {
		t.Fatalf("valid size: got (%d, %v), want (1024, nil)", got, err)
	}
	if _, err := checkedPeerMessageSize(1025, 1024); !errors.Is(err, types.ErrOversizedMessage) {
		t.Fatalf("configured limit: got %v, want %v", err, types.ErrOversizedMessage)
	}
	if _, err := checkedPeerMessageSize(maxInt+1, ^uint64(0)); !errors.Is(err, types.ErrOversizedMessage) {
		t.Fatalf("platform limit: got %v, want %v", err, types.ErrOversizedMessage)
	}
}
