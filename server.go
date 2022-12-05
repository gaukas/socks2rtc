package socks2rtc

import (
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/gaukas/socks2rtc/internal/utils"
	"github.com/gaukas/transportc"
)

type Server struct {
	// Config specifies the configuration for the underlying transportc.Listener.
	// If set to nil, use a default configuration.
	Config *transportc.Config

	// ListenAddr specifies the IP address to listen on.
	ListenAddr string

	rtcListener *transportc.Listener

	activeConnCnt atomic.Int64
}

// Start starts the server.
func (s *Server) Start() error {
	var err error
	s.rtcListener, err = s.Config.NewListener()
	if err != nil {
		return err
	}

	err = s.rtcListener.Start()
	if err != nil {
		return err
	}

	go s.serverloop()

	return nil
}

func (s *Server) Stop() error {
	return s.rtcListener.Stop()
}

func (s *Server) serverloop() {
	for {
		conn, err := s.rtcListener.Accept()
		if err != nil {
			log.Println("Server: Accept error: ", err)
			s.rtcListener.Stop()
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	// Read the request
	req, err := ReadRequest(conn)
	if err != nil {
		return
	}

	// increment the counter
	var currentCnt int64 = s.activeConnCnt.Load()
	for !s.activeConnCnt.CompareAndSwap(currentCnt, currentCnt+1) {
		currentCnt = s.activeConnCnt.Load()
	}

	switch req.Command {
	case CONNECT:
		log.Printf("CONNECT %s %s, %d connections alive", req.NetworkType, req.Address, currentCnt+1)
		s.handleConnect(conn, req)
	default:
		return // TODO: handle other commands
	}

	// decrement the counter
	currentCnt = s.activeConnCnt.Load()
	for !s.activeConnCnt.CompareAndSwap(currentCnt, currentCnt-1) {
		currentCnt = s.activeConnCnt.Load()
	}
}

func (s *Server) handleConnect(clientConn net.Conn, req *Request) {
	// Dial the remote peer
	proxyTargetConn, err := net.Dial(req.NetworkType, req.Address)
	if err != nil {
		return
	}
	defer proxyTargetConn.Close()

	// Build the response
	resp := &Response{
		Command:     CONNECT,
		NetworkType: req.NetworkType,
		Address:     proxyTargetConn.LocalAddr().String(),
	}

	// Write the response
	if _, err := resp.WriteTo(clientConn); err != nil {
		return
	}

	// Pipe the data for up to 15 minutes
	clientConn.SetDeadline(time.Now().Add(15 * time.Minute))
	proxyTargetConn.SetDeadline(time.Now().Add(15 * time.Minute))
	utils.FullPipe(clientConn, proxyTargetConn)
}
