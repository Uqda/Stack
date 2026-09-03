package types

import (
	"context"
	"encoding/hex"
	"testing"
)

func TestPublicKeyNames(t *testing.T) {
	key := make([]byte, 32)
	key[31] = 1
	label := hex.EncodeToString(key)
	r := NewNameResolver(nil, "")
	for _, suffix := range []string{NameMappingSuffix, LegacyNameMappingSuffix} {
		_, ip, err := r.Resolve(context.Background(), label+suffix)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", suffix, err)
		}
		if ip == nil || !isUQDAAddress(ip) {
			t.Fatalf("Resolve(%s) returned %v", suffix, ip)
		}
	}
}

func TestMalformedPublicKeyName(t *testing.T) {
	r := NewNameResolver(nil, "")
	if _, _, err := r.Resolve(context.Background(), "not-a-key"+NameMappingSuffix); err == nil {
		t.Fatal("malformed public key name accepted")
	}
}
