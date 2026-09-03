package safebind

import "testing"

func TestValidateSOCKS(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		allowPublic bool
		wantErr     bool
	}{
		{"IPv4 loopback", "127.0.0.1:1080", false, false},
		{"IPv6 loopback", "[::1]:1080", false, false},
		{"localhost", "localhost:1080", false, false},
		{"Unix socket", "/tmp/uqda-stack.sock", false, false},
		{"wildcard host", ":1080", false, true},
		{"IPv4 wildcard", "0.0.0.0:1080", false, true},
		{"IPv6 wildcard", "[::]:1080", false, true},
		{"LAN address", "192.0.2.10:1080", false, true},
		{"explicit public", "0.0.0.0:1080", true, false},
		{"bad port", "127.0.0.1:70000", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSOCKS(tt.endpoint, tt.allowPublic)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSOCKS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
