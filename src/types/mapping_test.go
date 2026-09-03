package types

import (
	"net"
	"testing"
)

func TestParseMappingString(t *testing.T) {
	tests := []struct {
		value                       string
		firstAddress, secondAddress string
		firstPort, secondPort       int
		wantErr                     bool
	}{
		{value: "1234", firstPort: 1234, secondPort: 1234},
		{value: "1234:4321", firstPort: 1234, secondPort: 4321},
		{value: "1234:127.0.0.1:4321", firstPort: 1234, secondAddress: "127.0.0.1", secondPort: 4321},
		{value: "127.0.0.2:1234:127.0.0.1:4321", firstAddress: "127.0.0.2", firstPort: 1234, secondAddress: "127.0.0.1", secondPort: 4321},
		{value: "1234:[201:db8::1]:4321", firstPort: 1234, secondAddress: "201:db8::1", secondPort: 4321},
		{value: "[::1]:1234:[201:db8::1]:4321", firstAddress: "::1", firstPort: 1234, secondAddress: "201:db8::1", secondPort: 4321},
		{value: "0", wantErr: true},
		{value: "65536", wantErr: true},
		{value: "not-a-port", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			fa, fp, sa, sp, err := parseMappingString(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && (fa != tt.firstAddress || fp != tt.firstPort || sa != tt.secondAddress || sp != tt.secondPort) {
				t.Fatalf("got %q:%d -> %q:%d", fa, fp, sa, sp)
			}
		})
	}
}

func TestLocalMappingsDefaultToLoopback(t *testing.T) {
	var tcp TCPLocalMappings
	if err := tcp.Set("8080:[201:db8::1]:80"); err != nil {
		t.Fatal(err)
	}
	if !tcp[0].Listen.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("TCP default listener = %s", tcp[0].Listen.IP)
	}
	var udp UDPLocalMappings
	if err := udp.Set("5353:[201:db8::1]:53"); err != nil {
		t.Fatal(err)
	}
	if !udp[0].Listen.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("UDP default listener = %s", udp[0].Listen.IP)
	}
}

func TestLocalMappingsRequireUQDAAddress(t *testing.T) {
	var tcp TCPLocalMappings
	if err := tcp.Set("8080:[2001:db8::1]:80"); err == nil {
		t.Fatal("accepted IPv6 address outside 0200::/7")
	}
	if err := tcp.Set("8080:[201:db8::1]:80"); err != nil {
		t.Fatalf("rejected UQDA address: %v", err)
	}
}

func TestRemoteMappings(t *testing.T) {
	var tcp TCPRemoteMappings
	for _, value := range []string{"8080", "8080:80", "8080:127.0.0.1:80"} {
		if err := tcp.Set(value); err != nil {
			t.Fatalf("TCP mapping %q: %v", value, err)
		}
	}
	var udp UDPRemoteMappings
	for _, value := range []string{"5353", "5353:53", "5353:127.0.0.1:53"} {
		if err := udp.Set(value); err != nil {
			t.Fatalf("UDP mapping %q: %v", value, err)
		}
	}
}
