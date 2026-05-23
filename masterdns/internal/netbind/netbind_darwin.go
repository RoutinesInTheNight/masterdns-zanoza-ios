//go:build darwin || ios

package netbind

import (
	"context"
	"errors"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func dialUDPBound(network string, raddr *net.UDPAddr, ifname string, local net.IP) (*net.UDPConn, error) {
	var ifaceIndex int
	if ifname != "" {
		ni, err := net.InterfaceByName(ifname)
		if err != nil {
			return nil, err
		}
		ifaceIndex = ni.Index
	}

	d := net.Dialer{
		Control: func(_ string, _ string, c syscall.RawConn) error {
			if ifaceIndex == 0 {
				return nil
			}
			var setErr error
			ctrlErr := c.Control(func(fd uintptr) {
				if e := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, ifaceIndex); e != nil {
					setErr = e
					return
				}
				// IPv6 variant is best-effort on a v4 socket.
				_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, ifaceIndex)
			})
			if ctrlErr != nil {
				return ctrlErr
			}
			return setErr
		},
	}
	if local != nil {
		d.LocalAddr = &net.UDPAddr{IP: local}
	}

	conn, err := d.DialContext(context.Background(), network, raddr.String())
	if err != nil {
		return nil, err
	}
	udp, ok := conn.(*net.UDPConn)
	if !ok {
		_ = conn.Close()
		return nil, errors.New("netbind: dialer returned non-UDP connection")
	}
	return udp, nil
}
