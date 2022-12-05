package socks2rtc

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/gaukas/socks2rtc/internal/utils"
	"github.com/gaukas/socks5"
	"github.com/gaukas/transportc"
	"golang.org/x/net/context"
)

const (
	CONNECT = socks5.REQUEST_CMD_CONNECT
	BIND    = socks5.REQUEST_CMD_BIND
	UDP     = socks5.REQUEST_CMD_UDP_ASSOCIATE

	TCP  = socks5.REQUEST_ATYP_DOMAINNAME
	TCP4 = socks5.REQUEST_ATYP_IPV4
	TCP6 = socks5.REQUEST_ATYP_IPV6
)

type Socks5Proxy struct {
	// Config specifies the configuration for the underlying transportc.Dialer.
	// If set to nil, use a default configuration.
	Config *transportc.Config

	// Timeout specifies a time limit for dialing to the remote peer.
	// If zero, no timeout is set.
	Timeout time.Duration

	createDialerOnce sync.Once // only create dialer once
	rtcDialer        *transportc.Dialer
	dialerMutex      sync.RWMutex
}

func (p *Socks5Proxy) dial(label string) (conn net.Conn, err error) {
	p.dialerMutex.Lock()
	defer p.dialerMutex.Unlock()

	p.createDialerOnce.Do(func() {
		p.rtcDialer, err = p.Config.NewDialer()
	})

	if err != nil {
		p.createDialerOnce = sync.Once{} // reset createDialerOnce
		// p.dialerMutex.Unlock()
		return nil, err
	}
	// p.dialerMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	if p.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, p.Timeout)
	}
	defer cancel()

	conn, err = p.rtcDialer.DialContext(ctx, label)
	if err != nil {
		return
	}

	conn.(*transportc.Conn).IdleKiller(p.Timeout)
	return
}

func (p *Socks5Proxy) Connect(dst net.Addr) (conn net.Conn, addr net.Addr, err error) {
	for {
		p.dialerMutex.RLock()
		dialerCopy := p.rtcDialer
		p.dialerMutex.RUnlock()

		conn, addr, err = p.connectOnce(dst)
		if err != nil { // dialer is broken, try to create a new one
			// TODO: log error
			// need new dialer ONLY if still using the same dialer
			p.dialerMutex.Lock()
			if dialerCopy == p.rtcDialer {
				log.Println("Assuming the old dialer is broken, creating a new one")
				p.createDialerOnce = sync.Once{}
				p.rtcDialer = nil
			}
			p.dialerMutex.Unlock()
			randSleep := rand.Intn(5) + 1 // skipcq: GSC-G404
			time.Sleep(time.Duration(randSleep*100) * time.Millisecond)
		} else {
			return
		}
	}
}

func (p *Socks5Proxy) connectOnce(dst net.Addr) (net.Conn, net.Addr, error) {
	conn, err := p.dial(fmt.Sprintf("%s-%s-%s", dst.Network(), dst.String(), utils.RandSeq(5)))
	if err != nil {
		return nil, nil, err
	}

	// Negotiate the SOCKS5 application
	req := &Request{
		Command:     CONNECT,
		NetworkType: dst.Network(),
		Address:     dst.String(),
	}

	if _, err := req.WriteTo(conn); err != nil {
		conn.Close()
		return nil, nil, err
	}

	// Read the response, must return in 2 seconds
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := ReadResponse(conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	conn.SetReadDeadline(time.Time{}) // disable deadline

	if resp.Command != CONNECT || resp.NetworkType != req.NetworkType {
		conn.Close()
		return nil, nil, fmt.Errorf("socks5: invalid response")
	}

	return &Conn{rtcConn: conn}, socks5.NewAddr(resp.NetworkType, resp.Address), nil
}

func (*Socks5Proxy) Bind(_ net.Addr) (chanConn chan net.Conn, chanAddr chan net.Addr, err error) {
	return nil, nil, socks5.ErrCommandNotSupported
}

func (*Socks5Proxy) UDPAssociate() (ua socks5.UDPAssociation, err error) {
	return nil, socks5.ErrCommandNotSupported
}
