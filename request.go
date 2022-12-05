package socks2rtc

import (
	"fmt"
	"io"

	"github.com/gaukas/socks5"
)

type Request struct {
	Command     byte
	NetworkType string
	Address     string
}

type Response = Request

var ReadResponse func(io.Reader) (*Response, error) = ReadRequest

func ReadRequest(r io.Reader) (*Request, error) {
	buf := make([]byte, 1024) // max size of a SOCKS5 request
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}

	if buf[0] != CONNECT && buf[0] != BIND && buf[0] != UDP {
		return nil, fmt.Errorf("invalid command: %d", buf[0])
	}

	req := &Request{
		Command: buf[0],
	}

	switch buf[1] {
	case TCP:
		req.NetworkType = "tcp"
	case TCP4:
		req.NetworkType = "tcp4"
	case TCP6:
		req.NetworkType = "tcp6"
	default:
		return nil, socks5.ErrConnNotAllowed
	}

	req.Address = string(buf[2:n])

	return req, nil
}

func (r *Request) WriteTo(w io.Writer) (int64, error) {
	var netType byte
	switch r.NetworkType {
	case "tcp":
		netType = TCP
	case "tcp4":
		netType = TCP4
	case "tcp6":
		netType = TCP6
	default:
		return 0, socks5.ErrNetworkUnreachable
	}

	req := []byte{r.Command, netType}
	req = append(req, []byte(r.Address)...)

	n, err := w.Write(req)
	return int64(n), err
}
