package socks2rtc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gaukas/socks2rtc/internal/utils"
	"github.com/gin-gonic/gin"
)

type WebSignalClient struct {
	// BaseURL is the base URL of the server, where offer will be sent and
	// answer will be received from.
	//
	// Not including the trailing slash or the protocol prefix (https://).
	BaseURL string // e.g. example.com:8443

	// UserID is the ID of the user.
	UserID uint64

	// Password is the password of the user used for HMAC authentication.
	Password []byte

	// TODO: security, anti-probing
}

// MakeOffer makes an offer and returns the offer ID.
//
//	POST https://example.com:8443/offer
//	{
//		offer:   <base64 encoded offer>,
//		sig:     <base64 encoded signature of offer raw bytes>,
//		pub_key: <base64 encoded public key>
//	}
//
// The response will be:
//
//	{
//		status: "success",
//		offer_id: <hex offer ID>
//	}
func (wsc *WebSignalClient) Offer(offer []byte) (offerID uint64, err error) {
	if wsc.BaseURL == "" {
		return 0, fmt.Errorf("empty BaseURL is not allowed")
	}

	mac := hmac.New(sha256.New, wsc.Password)
	mac.Write(offer)
	offerMacB64 := utils.ToBase64(mac.Sum(nil))
	offerB64 := utils.ToBase64(offer)

	postForm := map[string][]string{
		"offer": {string(offerB64)},
		"mac":   {string(offerMacB64)},
		"uid":   {fmt.Sprintf("%x", wsc.UserID)},
	}

	// send offer to server
	httpStatus, resp, err := utils.HttpsPost(
		"https://"+wsc.BaseURL+"/offer",
		postForm,
	)
	if err != nil {
		return 0, err
	}

	// parse response
	var respMap map[string]interface{}
	if json.Unmarshal(resp, &respMap) != nil {
		return 0, fmt.Errorf("failed to parse JSON response, HTTP status: %d, response: %s", httpStatus, resp)
	}

	if status, ok := respMap["status"]; ok {
		if status.(string) != "success" {
			return 0, fmt.Errorf("failed to send offer, status: %s, HTTP status: %d, response: %s", status, httpStatus, resp)
		} else {
			if offerID, ok := respMap["offer_id"]; ok {
				i, err := strconv.ParseUint(offerID.(string), 16, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse offer ID, error: %s", err)
				}
				return i, nil
			} else {
				return 0, fmt.Errorf("failed to get offer ID from response")
			}
		}
	} else {
		return 0, fmt.Errorf("failed to get status from response, response: %s", resp)
	}
}

// GetAnswer gets the answer from the server.
//
//	GET https://example.com:8443/answer?offer_id=<hex offer ID>&uid=<hex user ID>
//
// The response will be:
//
//	{
//		status: "success",
//		answer: <base64 encoded answer>
//	}
func (wsc *WebSignalClient) ReadAnswer(offerID uint64) (answer []byte, err error) {
	if wsc.BaseURL == "" {
		return nil, fmt.Errorf("empty AnswerFromURL is not allowed")
	}

	for {
		// get answer from server
		httpStatus, resp, err := utils.HttpsGet(
			fmt.Sprintf(
				"https://%s/answer?oid=%s&uid=%s",
				wsc.BaseURL,
				fmt.Sprintf("%x", offerID),
				fmt.Sprintf("%x", wsc.UserID),
			),
		)
		if err != nil {
			return nil, err
		}

		// parse response
		var respMap map[string]interface{}
		if json.Unmarshal(resp, &respMap) != nil {
			return nil, fmt.Errorf("failed to parse JSON response, HTTP status: %d, response: %s", httpStatus, resp)
		}

		if status, ok := respMap["status"]; ok {
			if status.(string) != "success" {
				if status.(string) == "pending" { // server yet to generate answer
					time.Sleep(2 * time.Second)
					continue
				}
				// server don't know the offer ID
				return nil, fmt.Errorf("failed to get answer, status: %s, HTTP status: %d, response: %s", status, httpStatus, resp)
			} else {
				if answerB64, ok := respMap["answer"].(string); ok {
					answer, err := utils.FromBase64([]byte(answerB64))
					if err != nil {
						return nil, fmt.Errorf("failed to decode answer from base64, error: %s", err)
					}
					return answer, nil
				} else {
					return nil, fmt.Errorf("failed to get answer from response")
				}
			}
		} else {
			return nil, fmt.Errorf("failed to get status from response, response: %s", resp)
		}
	}
}

// GetOffer isn't implemented by WebSignalClient.
func (*WebSignalClient) ReadOffer() (offerID uint64, offerBody []byte, err error) {
	return 0, nil, fmt.Errorf("not implemented by WebSignalClient")
}

// Answer isn't implemented by WebSignalClient.
func (*WebSignalClient) Answer(_ uint64, _ []byte) error {
	return fmt.Errorf("not implemented by WebSignalClient")
}

type incomingOffer struct {
	offer []byte // `json:"offer"` // Base64 encoded offer
	mac   []byte // `json:"mac"`   // Base64 encoded hmac of offer
	uid   uint64 // `json:"uid"`   // Hex representation of user ID
}

type receivedOffer struct {
	offerID uint64
	offer   []byte
}

type WebAnswer struct {
	Expiry   time.Time `json:"expiry"`
	OwnerUID uint64    `json:"owner_uid"`
	Answer   string    `json:"answer"`
}

type WebSignalServer struct {
	// offerQueue is a queue of offers to be answered.
	offerQueue chan receivedOffer

	// answerMap is a map of answers to be sent to clients.
	answerMap           map[uint64]WebAnswer
	answerMutex         sync.Mutex
	answerValidFor      time.Duration
	answerCleanerTicker *time.Ticker

	// authentication
	passwords   map[uint64][]byte
	passwdMutex sync.RWMutex

	// web server
	ginServer          *gin.Engine
	antiProbingHandler gin.HandlerFunc // this middleware is used to handle failed signal attempts
}

// NewWebSignalServer creates a new WebSignalServer.
// passwords is a map of user ID to password.
// antiProbe
func NewWebSignalServer(passwords map[uint64][]byte, antiProbeHandler gin.HandlerFunc) (*WebSignalServer, error) {
	if len(passwords) == 0 {
		return nil, fmt.Errorf("empty public keys")
	}

	wss := &WebSignalServer{
		offerQueue:         make(chan receivedOffer, 1024),
		answerMap:          make(map[uint64]WebAnswer),
		passwords:          passwords,
		antiProbingHandler: antiProbeHandler,
	}

	wss.answerValidFor = 60 * time.Second
	wss.answerCleanerTicker = time.NewTicker(5 * time.Second)
	go wss.startAnswerCleaner()

	// GIN-Gonic web server
	gin.SetMode(gin.ReleaseMode)
	wss.ginServer = gin.New()
	wss.ginServer.Use(gin.Recovery())
	wss.ginServer.POST("/offer", wss.offerHandler)
	wss.ginServer.GET("/answer", wss.answerHandler)
	if antiProbeHandler == nil {
		wss.antiProbingHandler = func(c *gin.Context) {
			c.AbortWithStatus(http.StatusNotFound) // TODO: this might  be fingerprintable
		}
	}
	wss.ginServer.NoMethod(wss.antiProbingHandler)
	wss.ginServer.NoRoute(wss.antiProbingHandler)

	return wss, nil
}

func (wss *WebSignalServer) Listen(addr string) {
	go wss.ginServer.Run(addr)
	time.Sleep(2 * time.Second)
}

func (wss *WebSignalServer) ReadOffer() (offerID uint64, offerBody []byte, err error) {
	offer := <-wss.offerQueue
	return offer.offerID, offer.offer, nil
}

func (wss *WebSignalServer) Answer(offerID uint64, answer []byte) error {
	wss.answerMutex.Lock()
	defer wss.answerMutex.Unlock()

	answerObj, ok := wss.answerMap[offerID]
	if !ok {
		return fmt.Errorf("offer ID not found")
	}

	answerObj.Answer = string(answer)
	wss.answerMap[offerID] = answerObj // update answer

	return nil
}

func (*WebSignalServer) ReadAnswer(_ uint64) (answer []byte, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (*WebSignalServer) Offer(_ []byte) (offerID uint64, err error) {
	return 0, fmt.Errorf("not implemented")
}

func (wss *WebSignalServer) AddPassword(uid uint64, password []byte) {
	wss.passwdMutex.Lock()
	defer wss.passwdMutex.Unlock()

	wss.passwords[uid] = password
}

func (wss *WebSignalServer) RemovePassword(uid uint64) {
	wss.passwdMutex.Lock()
	defer wss.passwdMutex.Unlock()

	delete(wss.passwords, uid)
}

func (wss *WebSignalServer) offerHandler(c *gin.Context) {
	// Parse POST form
	var offer incomingOffer
	var err error
	offer.offer, err = utils.FromBase64([]byte(c.PostForm("offer")))
	if err != nil {
		wss.antiProbingHandler(c)
		return
	}
	offer.mac, err = utils.FromBase64([]byte(c.PostForm("mac")))
	if err != nil {
		wss.antiProbingHandler(c)
		return
	}
	offer.uid, err = strconv.ParseUint(c.PostForm("uid"), 16, 64)
	if err != nil {
		wss.antiProbingHandler(c)
		return
	}

	wss.passwdMutex.RLock()
	defer wss.passwdMutex.RUnlock()
	// check if uid is in passwords
	var password []byte
	var ok bool
	if password, ok = wss.passwords[offer.uid]; !ok {
		wss.antiProbingHandler(c)
		return
	}

	// check mac
	mac := hmac.New(sha256.New, password)
	mac.Write(offer.offer)
	if !hmac.Equal(mac.Sum(nil), offer.mac) {
		wss.antiProbingHandler(c)
		return
	}

	// generate offer ID
	offerID := rand.Uint64() // skipcq: GSC-G404
	// push to offer queue
	wss.offerQueue <- receivedOffer{
		offerID: offerID,
		offer:   offer.offer,
	}
	// add empty answer to answer map
	wss.answerMutex.Lock()
	wss.answerMap[offerID] = WebAnswer{
		Expiry:   time.Now().Add(wss.answerValidFor),
		OwnerUID: offer.uid,
	}
	wss.answerMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"offer_id": fmt.Sprintf("%x", offerID),
	})
}

func (wss *WebSignalServer) answerHandler(c *gin.Context) {
	offerID := c.Query("oid")
	if offerID == "" {
		// anti-probing
		wss.antiProbingHandler(c)
		return
	}
	userID := c.Query("uid")
	if userID == "" {
		// anti-probing
		wss.antiProbingHandler(c)
		return
	}

	offerIDUint, err := strconv.ParseUint(offerID, 16, 64)
	if err != nil {
		// anti-probing
		wss.antiProbingHandler(c)
		return
	}

	ownerIDUint, err := strconv.ParseUint(userID, 16, 64)
	if err != nil {
		// anti-probing
		wss.antiProbingHandler(c)
		return
	}

	// check if answer exists
	wss.answerMutex.Lock()
	defer wss.answerMutex.Unlock()

	answer, ok := wss.answerMap[offerIDUint]
	if !ok || answer.OwnerUID != ownerIDUint {
		// anti-probing
		wss.antiProbingHandler(c)
		return
	}

	// check if answer is in
	if answer.Answer == "" {
		// return pending
		c.JSON(http.StatusOK, gin.H{
			"status": "pending",
		})
		return
	}

	// return answer
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"answer": string(utils.ToBase64([]byte(answer.Answer))),
	})

	// delete answer from map
	delete(wss.answerMap, offerIDUint)
}

func (wss *WebSignalServer) startAnswerCleaner() {
	for range wss.answerCleanerTicker.C {
		wss.answerMutex.Lock()
		for k, v := range wss.answerMap {
			if time.Now().After(v.Expiry) {
				delete(wss.answerMap, k)
			}
		}
		wss.answerMutex.Unlock()
	}
}
