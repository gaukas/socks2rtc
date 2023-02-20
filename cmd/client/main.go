package main

import (
	"flag"
	"log"
	"time"

	"github.com/gaukas/socks2rtc"
	"github.com/gaukas/socks5"
	"github.com/gaukas/transportc"
	"github.com/pion/webrtc/v3"
)

func main() {
	configFile := flag.String("config", "", "config file")
	flag.Parse()

	if *configFile == "" {
		flag.Usage()
		log.Fatal("config file is required")
	}

	conf, err := loadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	log.Printf("Using config file: %s\n", *configFile)

	webSignalClient := &socks2rtc.WebSignalClient{
		BaseURL:  conf.SignalBaseURL,
		UserID:   conf.UserID,
		Password: []byte(conf.Password),
	}

	socks5proxy := &socks2rtc.Socks5Proxy{
		Config: &transportc.Config{
			Signal: webSignalClient,
			WebRTCConfiguration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: conf.ICEServer.URLs,
					},
				},
			},
		},
		Timeout: time.Duration(conf.Timeout) * time.Second,
	}

	socks5Server, err := socks5.NewServer(nil, socks5proxy)
	if err != nil {
		panic(err)
	}

	err = socks5Server.Listen("tcp", conf.LocalAddress)
	if err != nil {
		panic(err)
	}

	select {}
}
