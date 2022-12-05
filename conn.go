package socks2rtc

import (
	"net"
	"time"
)

const (
	MAX_MSG_SIZE = 65535
)

// Conn implements a net.Conn interface
// Currently it is a wrapper around transportc.Conn.
//
// Keep for future.
type Conn struct {
	rtcConn net.Conn // transportc.Conn
}

// Read implements the net.Conn Read method.
func (c *Conn) Read(p []byte) (int, error) {
	return c.rtcConn.Read(p)
}

// Write implements the net.Conn Write method.
func (c *Conn) Write(p []byte) (int, error) {
	return c.rtcConn.Write(p)
}

// Close implements the net.Conn Close method.
func (c *Conn) Close() error {
	return c.rtcConn.Close()
}

// LocalAddr implements the net.Conn LocalAddr method.
func (c *Conn) LocalAddr() net.Addr {
	return c.rtcConn.LocalAddr()
}

// RemoteAddr implements the net.Conn RemoteAddr method.
func (c *Conn) RemoteAddr() net.Addr {
	return c.rtcConn.RemoteAddr()
}

// SetDeadline implements the net.Conn SetDeadline method.
// It sets both read and write deadlines in a single call.
//
// See SetReadDeadline and SetWriteDeadline for the behavior of the deadlines.
func (c *Conn) SetDeadline(deadline time.Time) error {
	return c.rtcConn.SetDeadline(deadline)
}

// SetReadDeadline implements the net.Conn SetReadDeadline method.
// It sets the read deadline for the connection.
//
// If the deadline is reached, Read will return an error.
// If the deadline is zero, Read will not time out.
func (c *Conn) SetReadDeadline(deadline time.Time) error {
	return c.rtcConn.SetReadDeadline(deadline)
}

// SetWriteDeadline implements the net.Conn SetWriteDeadline method.
// It sets the write deadline for the connection.
//
// If the deadline is reached, Write will return an error.
// If the deadline is zero, Write will not time out.
func (c *Conn) SetWriteDeadline(deadline time.Time) error {
	return c.rtcConn.SetWriteDeadline(deadline)
}
