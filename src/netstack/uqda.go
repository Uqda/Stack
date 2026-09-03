package netstack

import (
	"log"
	"net"

	"github.com/Uqda/Core/src/core"
	"github.com/Uqda/Core/src/ipv6rwc"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

type UQDANIC struct {
	stack      *UQDANetstack
	ipv6rwc    *ipv6rwc.ReadWriteCloser
	dispatcher stack.NetworkDispatcher
	readBuf    []byte
	writeBuf   []byte
	rstPackets chan *stack.PacketBuffer
}

func (s *UQDANetstack) NewUQDANIC(ygg *core.Core) tcpip.Error {
	rwc := ipv6rwc.NewReadWriteCloser(ygg)
	mtu := rwc.MTU()
	nic := &UQDANIC{
		ipv6rwc:    rwc,
		readBuf:    make([]byte, mtu),
		writeBuf:   make([]byte, mtu),
		rstPackets: make(chan *stack.PacketBuffer, 100),
	}
	if err := s.stack.CreateNIC(1, nic); err != nil {
		return err
	}
	go func() {
		var rx int
		var err error
		for {
			rx, err = nic.ipv6rwc.Read(nic.readBuf)
			if err != nil {
				log.Println(err)
				break
			}
			pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{
				Payload: buffer.MakeWithData(nic.readBuf[:rx]),
			})
			nic.dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pkb)
			pkb.DecRef()
		}
	}()
	go func() {
		for {
			pkt := <-nic.rstPackets
			if pkt == nil {
				continue
			}
			_ = nic.writePacket(pkt)
			pkt.DecRef()
		}
	}()
	_, snet, err := net.ParseCIDR("0200::/7")
	if err != nil {
		return &tcpip.ErrBadAddress{}
	}
	subnet, err := tcpip.NewSubnet(
		tcpip.AddrFromSlice(snet.IP.To16()),
		tcpip.MaskFrom(string(snet.Mask)),
	)
	if err != nil {
		return &tcpip.ErrBadAddress{}
	}
	s.stack.AddRoute(tcpip.Route{
		Destination: subnet,
		NIC:         1,
	})
	if s.stack.HandleLocal() {
		ip := ygg.Address()
		if err := s.stack.AddProtocolAddress(
			1,
			tcpip.ProtocolAddress{
				Protocol:          ipv6.ProtocolNumber,
				AddressWithPrefix: tcpip.AddrFromSlice(ip.To16()).WithPrefix(),
			},
			stack.AddressProperties{},
		); err != nil {
			return err
		}
	}
	return nil
}

func (e *UQDANIC) Attach(dispatcher stack.NetworkDispatcher) { e.dispatcher = dispatcher }

func (e *UQDANIC) IsAttached() bool { return e.dispatcher != nil }

func (e *UQDANIC) MTU() uint32 { return uint32(e.ipv6rwc.MTU()) }

func (e *UQDANIC) SetMTU(uint32) {}

func (*UQDANIC) Capabilities() stack.LinkEndpointCapabilities { return stack.CapabilityNone }

func (*UQDANIC) MaxHeaderLength() uint16 { return 40 }

func (*UQDANIC) LinkAddress() tcpip.LinkAddress { return "" }

func (*UQDANIC) SetLinkAddress(tcpip.LinkAddress) {}

func (*UQDANIC) Wait() {}

func (e *UQDANIC) writePacket(
	pkt *stack.PacketBuffer,
) tcpip.Error {
	// We need to recover from panic() here because
	// parser in ToView() gets confused on some packets
	// without payload and panics
	defer func() {
		r := recover()
		if r != nil {
		}
	}()
	vv := pkt.ToView()
	n, err := vv.Read(e.writeBuf)
	if err != nil {
		return &tcpip.ErrAborted{}
	}
	_, err = e.ipv6rwc.Write(e.writeBuf[:n])
	if err != nil {
		return &tcpip.ErrAborted{}
	}
	return nil
}

func (e *UQDANIC) WritePackets(
	list stack.PacketBufferList,
) (int, tcpip.Error) {
	var i int = 0
	var err tcpip.Error = nil
	for i, pkt := range list.AsSlice() {
		if pkt.Data().Size() == 0 {
			if pkt.Network().TransportProtocol() == tcp.ProtocolNumber {
				tcpHeader := header.TCP(pkt.TransportHeader().Slice())
				if (tcpHeader.Flags() & header.TCPFlagRst) == header.TCPFlagRst {
					pkt.IncRef()
					select {
					case e.rstPackets <- pkt:
						// Packet queued successfully
					default:
						// Channel full, drop packet and release ref
						pkt.DecRef()
					}
					continue
				}
			}
		}
		err = e.writePacket(pkt)
		if err != nil {
			log.Println(err)
			return i - 1, err
		}
	}

	return i, nil
}

func (e *UQDANIC) WriteRawPacket(*stack.PacketBuffer) tcpip.Error {
	panic("not implemented")
}

func (*UQDANIC) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *UQDANIC) AddHeader(*stack.PacketBuffer) {
}

func (e *UQDANIC) ParseHeader(*stack.PacketBuffer) bool {
	return true
}

func (e *UQDANIC) Close() {
	e.stack.stack.RemoveNIC(1)
	e.dispatcher = nil
}

func (e *UQDANIC) SetOnCloseAction(func()) {}
