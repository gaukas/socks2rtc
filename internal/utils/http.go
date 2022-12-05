package utils

import (
	ctls "crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"

	tls "github.com/refraction-networking/utls"

	"golang.org/x/net/http2"
)

var HTTPS_INSECURE = false // turn on for debug (only!)

func IsHTTPS(url string) bool {
	// check if start with https://
	return strings.HasPrefix(url, "https://")
}

func HttpsGet(url string) (status int, body []byte, err error) {
	client := &http.Client{Transport: transport2()}

	res, err := client.Get(url)
	if err != nil {
		return res.StatusCode, nil, err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	return res.StatusCode, body, err
}

func HttpsPost(url string, values map[string][]string) (status int, body []byte, err error) {
	client := &http.Client{Transport: transport2()}

	res, err := client.PostForm(url, values)
	if err != nil {
		return res.StatusCode, nil, err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	return res.StatusCode, body, err
}

func transport2() *http2.Transport {
	return &http2.Transport{
		DialTLS:            utlsDial,
		DisableCompression: true,
		AllowHTTP:          false,
	}
}

func utlsDial(_, addr string, cconf *ctls.Config) (net.Conn, error) {
	tcpConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.UClient(tcpConn, &tls.Config{
		ServerName:         cconf.ServerName,
		InsecureSkipVerify: HTTPS_INSECURE, // TODO: verify certificate
		MinVersion:         tls.VersionTLS12,
	},
		tls.HelloChrome_106_Shuffle)

	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}

	return tlsConn, nil
}
