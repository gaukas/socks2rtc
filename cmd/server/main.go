package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/gaukas/socks2rtc"
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

	// translate all string passwords to []byte
	passwords := map[uint64][]byte{}
	for id, password := range conf.WebSignal.DefaultUsers {
		passwords[id] = []byte(password)
		log.Printf("User %d using password %s", id, password)
	}

	webSignalServer, err := socks2rtc.NewWebSignalServer(
		passwords,
		nil,
	)
	if err != nil {
		panic(err)
	}

	webSignalServer.Listen(conf.ServerAddr)

	server := &socks2rtc.Server{
		Config: &transportc.Config{
			SignalMethod: webSignalServer,
			WebRTCConfiguration: webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: conf.ICEServer.URLs,
					},
				},
			},
		},
		ListenAddr: conf.ServerAddr,
	}

	err = server.Start()
	if err != nil {
		panic(err)
	}
	log.Println("Server started. Now handling connections...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	log.Println("Shutting down...")
	server.Stop()
	log.Println("Bye!")
}
