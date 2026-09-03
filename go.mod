module github.com/Uqda/Stack

go 1.25.13

// Keep the serialized peer-debug snapshot fix used by UQDA Core v0.1.4.
replace github.com/Arceliar/ironwood => ./third_party/ironwood

require (
	github.com/gologme/log v1.3.0
	github.com/hashicorp/go-syslog v1.0.0
	github.com/hjson/hjson-go/v4 v4.6.0
	github.com/things-go/go-socks5 v0.1.0
	github.com/Uqda/Core v0.1.4
	gvisor.dev/gvisor v0.0.0-20250812171554-968e93457fe6
)

require (
	github.com/Arceliar/ironwood v0.0.0-20260613025018-d50055b11f5e // indirect
	github.com/Arceliar/phony v0.0.0-20220903101357-530938a4b13d // indirect
	github.com/bits-and-blooms/bitset v1.24.4 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.7.1 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/pprof v0.0.0-20241017200806-017d972448fc // indirect
	github.com/onsi/ginkgo/v2 v2.20.2 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	go.uber.org/mock v0.5.2 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
)
