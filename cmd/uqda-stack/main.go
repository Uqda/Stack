package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gologme/log"
	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/hjson/hjson-go/v4"
	"github.com/things-go/go-socks5"

	"github.com/Uqda/Core/src/address"
	"github.com/Uqda/Core/src/config"
	"github.com/Uqda/Core/src/core"
	"github.com/Uqda/Core/src/multicast"
	"github.com/Uqda/Core/src/version"

	"github.com/Uqda/Stack/src/netstack"
	"github.com/Uqda/Stack/src/safebind"
	"github.com/Uqda/Stack/src/termui"
	"github.com/Uqda/Stack/src/types"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

type node struct {
	core       *core.Core
	multicast  *multicast.Multicast
	socks5Tcp  net.Listener
	socks5Unix net.Listener
}

type UDPSession struct {
	conn       interface{}
	remoteAddr net.Addr
}

// The main function is responsible for configuring and starting UQDA.
func main() {
	var localtcp types.TCPLocalMappings
	var localudp types.UDPLocalMappings
	var remotetcp types.TCPRemoteMappings
	var remoteudp types.UDPRemoteMappings
	genconf := flag.Bool("genconf", false, "print a new config to stdout")
	useconf := flag.Bool("useconf", false, "read HJSON/JSON config from stdin")
	useconffile := flag.String("useconffile", "", "read HJSON/JSON config from specified file path")
	normaliseconf := flag.Bool("normaliseconf", false, "use in combination with either -useconf or -useconffile, outputs your configuration normalised")
	exportkey := flag.Bool("exportkey", false, "use in combination with either -useconf or -useconffile, outputs your private key in PEM format")
	confjson := flag.Bool("json", false, "print configuration from -genconf or -normaliseconf as JSON instead of HJSON")
	autoconf := flag.Bool("autoconf", false, "automatic mode (dynamic IP, peer with IPv6 neighbors)")
	ver := flag.Bool("version", false, "prints the version of this build")
	logto := flag.String("logto", "stdout", "file path to log to, \"syslog\" or \"stdout\"")
	getaddr := flag.Bool("address", false, "use in combination with either -useconf or -useconffile, outputs your IPv6 address")
	getsnet := flag.Bool("subnet", false, "use in combination with either -useconf or -useconffile, outputs your IPv6 subnet")
	getpkey := flag.Bool("publickey", false, "use in combination with either -useconf or -useconffile, outputs your public key")
	checkOnly := flag.Bool("check", false, "validate configuration and listener policy without starting the node")
	loglevel := flag.String("loglevel", "info", "loglevel to enable")
	socks := flag.String("socks", "", "local SOCKS5 listener, e.g. 127.0.0.1:1080, [::1]:1080, or /tmp/uqda-stack.sock")
	allowPublicSOCKS := flag.Bool("allow-public-socks", false, "allow a non-loopback SOCKS listener (dangerous; secure it with a firewall)")
	colorMode := flag.String("color", "auto", "terminal colors: auto, always, or never")
	noColor := flag.Bool("no-color", false, "disable terminal colors (equivalent to -color=never)")
	nameserver := flag.String("nameserver", "", "the UQDA IPv6 address to use as a DNS server for SOCKS")
	flag.Var(&localtcp, "local-tcp", "TCP ports to forward to a remote UQDA node, e.g. 127.0.0.1:8080:[201:db8::1]:80")
	flag.Var(&localudp, "local-udp", "UDP ports to forward to the remote UQDA node, e.g. 22:[a:b:c:d]:2022, 127.0.0.1:[a:b:c:d]:22")
	flag.Var(&remotetcp, "remote-tcp", "TCP ports to expose to the network, e.g. 22, 2022:22, 22:192.168.1.1:2022")
	flag.Var(&remoteudp, "remote-udp", "UDP ports to expose to the network, e.g. 22, 2022:22, 22:192.168.1.1:2022")
	flag.Parse()
	if *noColor {
		*colorMode = "never"
	}
	mode, colorErr := termui.ParseMode(*colorMode)
	if colorErr != nil {
		fmt.Fprintln(os.Stderr, "uqda-stack:", colorErr)
		os.Exit(2)
	}
	ui := termui.New(os.Stdout, mode)
	if err := safebind.ValidateSOCKS(*socks, *allowPublicSOCKS); err != nil {
		fmt.Fprintln(os.Stderr, "uqda-stack:", err)
		os.Exit(2)
	}

	// Catch interrupts from the operating system to exit gracefully.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Create a new logger that logs output to stdout.
	var logger *log.Logger
	switch *logto {
	case "stdout":
		logger = log.New(os.Stdout, "", log.Flags())

	case "syslog":
		if syslogger, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "DAEMON", version.BuildName()); err == nil {
			logger = log.New(syslogger, "", log.Flags()&^(log.Ldate|log.Ltime))
		}

	default:
		if logfd, err := os.OpenFile(*logto, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); err == nil {
			logger = log.New(logfd, "", log.Flags())
		}
	}
	if logger == nil {
		logger = log.New(os.Stdout, "", log.Flags())
		logger.Warnln("Logging defaulting to stdout")
	}
	if *normaliseconf {
		setLogLevel("error", logger)
	} else {
		setLogLevel(*loglevel, logger)
	}

	cfg := config.GenerateConfig()
	var err error
	switch {
	case *ver:
		fmt.Println("Build name:", version.BuildName())
		fmt.Println("Build version:", version.BuildVersion())
		return

	case *autoconf:
		// Force AdminListen to none in uqda-stack
		cfg.AdminListen = "none"
		// Use an autoconf-generated config, this will give us random keys and
		// port numbers, and will use an automatically selected TUN interface.

	case *useconf:
		if _, err := cfg.ReadFrom(os.Stdin); err != nil {
			panic(err)
		}

	case *useconffile != "":
		f, err := os.Open(*useconffile)
		if err != nil {
			panic(err)
		}
		if _, err := cfg.ReadFrom(f); err != nil {
			panic(err)
		}
		_ = f.Close()

	case *genconf:
		// Force AdminListen to none in uqda-stack
		cfg.AdminListen = "none"
		var bs []byte
		if *confjson {
			bs, err = json.MarshalIndent(cfg, "", "  ")
		} else {
			bs, err = hjson.Marshal(cfg)
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(string(bs))
		return

	default:
		fmt.Println("Usage:")
		flag.PrintDefaults()

		if *getaddr || *getsnet {
			fmt.Println("\nError: You need to specify some config data using -useconf or -useconffile.")
		}
		return
	}
	// UQDA Stack is deliberately unprivileged and must not collide with a
	// system-wide UQDA Core administration endpoint.
	cfg.AdminListen = "none"

	privateKey := ed25519.PrivateKey(cfg.PrivateKey)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	switch {
	case *getaddr:
		addr := address.AddrForKey(publicKey)
		ip := net.IP(addr[:])
		fmt.Println(ip.String())
		return

	case *getsnet:
		snet := address.SubnetForKey(publicKey)
		ipnet := net.IPNet{
			IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
			Mask: net.CIDRMask(len(snet)*8, 128),
		}
		fmt.Println(ipnet.String())
		return

	case *getpkey:
		fmt.Println(hex.EncodeToString(publicKey))
		return

	case *normaliseconf:
		cfg.AdminListen = "none"
		if cfg.PrivateKeyPath != "" {
			cfg.PrivateKey = nil
		}
		var bs []byte
		if *confjson {
			bs, err = json.MarshalIndent(cfg, "", "  ")
		} else {
			bs, err = hjson.Marshal(cfg)
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(string(bs))
		return

	case *exportkey:
		pem, err := cfg.MarshalPEMPrivateKey()
		if err != nil {
			panic(err)
		}
		fmt.Println(string(pem))
		return

	case *checkOnly:
		checkedAddr := address.AddrForKey(publicKey)
		ui.Heading("UQDA Stack configuration check")
		ui.Success("config", "HJSON/JSON parsed successfully")
		ui.Success("identity", net.IP(checkedAddr[:]).String())
		ui.Success("admin", "disabled in userspace mode")
		if *socks != "" {
			if *allowPublicSOCKS {
				ui.Warning("SOCKS5", *socks+" (public override enabled)")
			} else {
				ui.Success("SOCKS5", *socks)
			}
		}
		return
	}

	n := &node{}

	// Setup the UQDA node itself.
	{
		options := []core.SetupOption{
			core.NodeInfo(cfg.NodeInfo),
			core.NodeInfoPrivacy(cfg.NodeInfoPrivacy),
		}
		for _, addr := range cfg.Listen {
			options = append(options, core.ListenAddress(addr))
		}
		for _, peer := range cfg.Peers {
			options = append(options, core.Peer{URI: peer})
		}
		for intf, peers := range cfg.InterfacePeers {
			for _, peer := range peers {
				options = append(options, core.Peer{URI: peer, SourceInterface: intf})
			}
		}
		for _, allowed := range cfg.AllowedPublicKeys {
			k, err := hex.DecodeString(allowed)
			if err != nil {
				panic(err)
			}
			options = append(options, core.AllowedPublicKey(k[:]))
		}
		if n.core, err = core.New(cfg.Certificate, logger, options...); err != nil {
			panic(err)
		}
		address, subnet := n.core.Address(), n.core.Subnet()
		publicstr := hex.EncodeToString(n.core.PublicKey())
		logger.Printf("Your public key is %s", publicstr)
		logger.Printf("Your IPv6 address is %s", address.String())
		logger.Printf("Your IPv6 subnet is %s", subnet.String())
		logger.Printf("Your UQDA Stack resolver name is %s%s", publicstr, types.NameMappingSuffix)
		ui.Heading("UQDA Stack")
		ui.Success("identity", address.String())
		ui.Success("privileges", "userspace mode; TUN/root not required")
	}

	// Setup the multicast module.
	{
		options := []multicast.SetupOption{}
		for _, intf := range cfg.MulticastInterfaces {
			options = append(options, multicast.MulticastInterface{
				Regex:    regexp.MustCompile(intf.Regex),
				Beacon:   intf.Beacon,
				Listen:   intf.Listen,
				Port:     intf.Port,
				Priority: uint8(intf.Priority),
				Password: intf.Password,
			})
		}
		if n.multicast, err = multicast.New(n.core, logger, options...); err != nil {
			panic(err)
		}
	}

	// Setup UQDA netstack
	s, err := netstack.CreateUQDANetstack(n.core)
	if err != nil {
		panic(err)
	}

	// Create SOCKS server
	{
		if socks != nil && *socks != "" {
			socksOptions := []socks5.Option{
				socks5.WithDial(s.DialContext),
			}
			var resolver *types.NameResolver = nil
			if nameserver != nil && *nameserver != "" {
				resolver = types.NewNameResolver(s, *nameserver)
			} else {
				logger.Infof("DNS nameserver is not set!")
				logger.Infof("SOCKS server will resolve only .pk.uqda/.pk.ygg names without -nameserver")
				resolver = types.NewNameResolver(s, "")
			}
			socksOptions = append(socksOptions, socks5.WithResolver(resolver))
			if logger.GetLevel("debug") {
				socksOptions = append(socksOptions, socks5.WithLogger(logger))
			}
			server := socks5.NewServer(socksOptions...)
			if strings.Contains(*socks, ":") {
				logger.Infof("Starting SOCKS server on %s", *socks)
				n.socks5Tcp, err = net.Listen("tcp", *socks)
				if err != nil {
					panic(err)
				}
				go func() {
					if serveErr := server.Serve(n.socks5Tcp); serveErr != nil && ctx.Err() == nil {
						logger.Errorf("SOCKS5 listener stopped: %v", serveErr)
					}
				}()
				if *allowPublicSOCKS {
					ui.Warning("SOCKS5", *socks+" (non-loopback listener explicitly enabled)")
				} else {
					ui.Success("SOCKS5", *socks)
				}
			} else {
				logger.Infof("Starting SOCKS server with socket file %s", *socks)
				n.socks5Unix, err = net.Listen("unix", *socks)
				if err != nil {
					// If address in use, try connecting to
					// the socket to see if other uqda-stack
					// instance is listening on it

					if isErrorAddressAlreadyInUse(err) {
						probe, dialErr := net.DialTimeout("unix", *socks, time.Second)
						if dialErr != nil {
							// Unlink dead socket if not connected
							info, statErr := os.Lstat(*socks)
							if statErr != nil || info.Mode()&os.ModeSocket == 0 {
								panic(fmt.Errorf("refusing to remove non-socket path %q", *socks))
							}
							err = os.Remove(*socks)
							if err != nil {
								panic(err)
							}
							n.socks5Unix, err = net.Listen("unix", *socks)
							if err != nil {
								panic(err)
							}
						} else {
							_ = probe.Close()
							panic(fmt.Errorf("Another uqda-stack instance is listening on socket '%s'", *socks))
						}
					} else {
						panic(err)
					}
				}
				go func() {
					if serveErr := server.Serve(n.socks5Unix); serveErr != nil && ctx.Err() == nil {
						logger.Errorf("SOCKS5 Unix listener stopped: %v", serveErr)
					}
				}()
				if err := os.Chmod(*socks, 0600); err != nil {
					panic(fmt.Errorf("restrict SOCKS socket permissions: %w", err))
				}
				ui.Success("SOCKS5", *socks+" (mode 0600)")
			}
		}
	}

	// Create local TCP mappings (forwarding connections from local port
	// to remote UQDA node)
	{
		for _, mapping := range localtcp {
			go func(mapping types.TCPMapping) {
				listener, err := net.ListenTCP("tcp", mapping.Listen)
				if err != nil {
					panic(err)
				}
				logger.Infof("Mapping local TCP port %d to UQDA %s", mapping.Listen.Port, mapping.Mapped)
				for {
					c, err := listener.Accept()
					if err != nil {
						panic(err)
					}
					r, err := s.DialTCP(mapping.Mapped)
					if err != nil {
						logger.Errorf("Failed to connect to %s: %s", mapping.Mapped, err)
						_ = c.Close()
						continue
					}
					go types.ProxyTCP(n.core.MTU(), c, r)
				}
			}(mapping)
		}
	}

	// Create local UDP mappings (forwarding connections from local port
	// to remote UQDA node)
	{
		for _, mapping := range localudp {
			go func(mapping types.UDPMapping) {
				mtu := n.core.MTU()
				udpListenConn, err := net.ListenUDP("udp", mapping.Listen)
				if err != nil {
					panic(err)
				}
				logger.Infof("Mapping local UDP port %d to UQDA %s", mapping.Listen.Port, mapping.Mapped)
				localUdpConnections := new(sync.Map)
				udpBuffer := make([]byte, mtu)
				for {
					bytesRead, remoteUdpAddr, err := udpListenConn.ReadFrom(udpBuffer)
					if err != nil {
						if bytesRead == 0 {
							continue
						}
					}

					remoteUdpAddrStr := remoteUdpAddr.String()

					connVal, ok := localUdpConnections.Load(remoteUdpAddrStr)

					if !ok {
						logger.Debugf("Creating new session for %s", remoteUdpAddr.String())
						udpFwdConn, err := s.DialUDP(mapping.Mapped)
						if err != nil {
							logger.Errorf("Failed to connect to %s: %s", mapping.Mapped, err)
							continue
						}
						udpSession := &UDPSession{
							conn:       udpFwdConn,
							remoteAddr: remoteUdpAddr,
						}
						localUdpConnections.Store(remoteUdpAddrStr, udpSession)
						go types.ReverseProxyUDP(mtu, udpListenConn, remoteUdpAddr, udpFwdConn)
					}

					udpSession, ok := connVal.(*UDPSession)
					if !ok {
						continue
					}

					udpFwdConnPtr := udpSession.conn.(*gonet.UDPConn)
					udpFwdConn := *udpFwdConnPtr

					_, err = udpFwdConn.Write(udpBuffer[:bytesRead])
					if err != nil {
						logger.Debugf("Cannot write from uqda to udp listener: %q", err)
						udpFwdConn.Close()
						localUdpConnections.Delete(remoteUdpAddrStr)
						continue
					}
				}
			}(mapping)
		}
	}

	// Create remote TCP mappings (forwarding connections from UQDA
	// node to local port)
	{
		for _, mapping := range remotetcp {
			go func(mapping types.TCPMapping) {
				listener, err := s.ListenTCP(mapping.Listen)
				if err != nil {
					panic(err)
				}
				logger.Infof("Mapping UQDA TCP port %d to %s", mapping.Listen.Port, mapping.Mapped)
				for {
					c, err := listener.Accept()
					if err != nil {
						panic(err)
					}
					r, err := net.DialTCP("tcp", nil, mapping.Mapped)
					if err != nil {
						logger.Errorf("Failed to connect to %s: %s", mapping.Mapped, err)
						_ = c.Close()
						continue
					}
					go types.ProxyTCP(n.core.MTU(), c, r)
				}
			}(mapping)
		}
	}

	// Create remote UDP mappings (forwarding connections from UQDA
	// node to local port)
	{
		for _, mapping := range remoteudp {
			go func(mapping types.UDPMapping) {
				mtu := n.core.MTU()
				udpListenConn, err := s.ListenUDP(mapping.Listen)
				if err != nil {
					panic(err)
				}
				logger.Infof("Mapping UQDA UDP port %d to %s", mapping.Listen.Port, mapping.Mapped)
				remoteUdpConnections := new(sync.Map)
				udpBuffer := make([]byte, mtu)
				for {
					bytesRead, remoteUdpAddr, err := udpListenConn.ReadFrom(udpBuffer)
					if err != nil {
						logger.Debugf("udp readFrom error: %v", err)
					}
					if bytesRead == 0 {
						continue
					}

					remoteUdpAddrStr := remoteUdpAddr.String()

					var udpSession *UDPSession = nil

					connVal, ok := remoteUdpConnections.Load(remoteUdpAddrStr)

					if !ok {
						logger.Debugf("Creating new session for %s", remoteUdpAddr.String())
						udpFwdConn, err := net.DialUDP("udp", nil, mapping.Mapped)
						if err != nil {
							logger.Errorf("Failed to connect to %s: %s", mapping.Mapped, err)
							continue
						}
						udpSession = &UDPSession{
							conn:       udpFwdConn,
							remoteAddr: remoteUdpAddr,
						}
						remoteUdpConnections.Store(remoteUdpAddrStr, udpSession)
						go types.ReverseProxyUDP(mtu, udpListenConn, remoteUdpAddr, udpFwdConn)
					} else {
						udpSession, ok = connVal.(*UDPSession)

						if !ok {
							continue
						}
					}

					udpFwdConnPtr := udpSession.conn.(*net.UDPConn)
					udpFwdConn := *udpFwdConnPtr

					_, err = udpFwdConn.Write(udpBuffer[:bytesRead])
					if err != nil {
						logger.Debugf("Cannot write from uqda to udp listener: %q", err)
						udpFwdConn.Close()
						remoteUdpConnections.Delete(remoteUdpAddrStr)
						continue
					}
				}
			}(mapping)
		}
	}

	// Block until we are told to shut down.
	<-ctx.Done()

	// Shut down the node.
	if n.multicast != nil {
		_ = n.multicast.Stop()
	}
	if n.socks5Unix != nil {
		_ = n.socks5Unix.Close()
		_ = os.Remove(*socks)
		logger.Infof("Stopped SOCKS5 UNIX socket listener")
	}
	if n.socks5Tcp != nil {
		_ = n.socks5Tcp.Close()
		logger.Infof("Stopped SOCKS5 TCP listener")
	}
		n.core.Stop()
}

// Helper to detect if socket address is in use
// https://stackoverflow.com/a/52152912
func isErrorAddressAlreadyInUse(err error) bool {
	var eOsSyscall *os.SyscallError
	if !errors.As(err, &eOsSyscall) {
		return false
	}
	var errErrno syscall.Errno // doesn't need a "*" (ptr) because it's already a ptr (uintptr)
	if !errors.As(eOsSyscall, &errErrno) {
		return false
	}
	if errors.Is(errErrno, syscall.EADDRINUSE) {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}

// Helper to set logging level
func setLogLevel(loglevel string, logger *log.Logger) {
	levels := [...]string{"error", "warn", "info", "debug", "trace"}
	loglevel = strings.ToLower(loglevel)

	contains := func() bool {
		for _, l := range levels {
			if l == loglevel {
				return true
			}
		}
		return false
	}

	if !contains() { // set default log level
		logger.Infoln("Loglevel parse failed. Set default level(info)")
		loglevel = "info"
	}

	for _, l := range levels {
		logger.EnableLevel(l)
		if l == loglevel {
			break
		}
	}
}
