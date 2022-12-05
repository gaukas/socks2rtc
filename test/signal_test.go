package socks2rtc_test

import (
	"testing"

	"github.com/gaukas/socks2rtc"
	"github.com/gaukas/socks2rtc/internal/utils"
)

func TestWebSignal(t *testing.T) {
	t.Skip() // Skipped for automated testing, because it requires a running HTTPS server

	utils.HTTPS_INSECURE = true // debug flag to allow self-signed certs

	wsc := &socks2rtc.WebSignalClient{
		BaseURL:  "localhost",
		UserID:   123456789,
		Password: []byte("password"),
	}

	wss, err := socks2rtc.NewWebSignalServer(
		map[uint64][]byte{
			0:         []byte("12345678901234567890123456789012"),
			1:         []byte("12345678901234567890123456789012"),
			123456789: []byte("password"),
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	wss.Listen("127.0.0.1:29443")

	// MakeOffer should succeed
	helloID, err := wsc.MakeOffer([]byte("Hello, world!"))
	if err != nil {
		t.Fatal(err)
	}

	// GetOffer should succeed
	id, offer, err := wss.GetOffer()
	if err != nil {
		t.Fatal(err)
	}

	if id != helloID {
		t.Fatal("ID mismatch")
	}

	if string(offer) != "Hello, world!" {
		t.Fatal("Offer mismatch")
	}

	// Answer should succeed
	err = wss.Answer(id, []byte("Hey, you!"))
	if err != nil {
		t.Fatal(err)
	}

	// GetAnswer should succeed
	answer, err := wsc.GetAnswer(id)
	if err != nil {
		t.Fatal(err)
	}

	if string(answer) != "Hey, you!" {
		t.Fatalf("Answer mismatch, got %s", answer)
	}
}
