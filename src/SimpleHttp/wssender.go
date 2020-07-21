// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// HYCOWSSender is a simple websocket sending client
type HYCOWSSender interface {
	GetRelayWSURI(correlationID string) string
	CreateRelaySASToken() string
	ConnectRelayWS(sasToken string)
}

type hycoWSSender struct {
	ns      string
	path    string
	keyrule string
	key     string
}

func main() {
	log.SetFlags(0)

	var client HYCOWSSender
	client = hycoWSSender{
		ns:      "gorelay.servicebus.windows.net",
		path:    "yesclientauth",
		keyrule: "managepolicy",
		key:     "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE="}

	sasToken := client.CreateRelaySASToken()
	client.ConnectRelayWS(sasToken)
}

func (sender hycoWSSender) ConnectRelayWS(sasToken string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := sender.GetRelayWSURI("")
	log.Printf("connecting to %s", u)

	header := http.Header{}
	header["ServiceBusAuthorization"] = []string{sasToken}
	c, _, err := websocket.DefaultDialer.Dial(u, header)

	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func (c hycoWSSender) GetRelayWSURI(correlationID string) string {
	query := "sb-hc-action=connect"
	if correlationID != "" {
		query = "&sb-hc-id=" + correlationID
	}

	u := url.URL{Scheme: "wss", Host: c.ns + ":443", Path: "$hc/" + c.path, RawQuery: query}
	fmt.Println(u.String())
	return u.String()
}

func (c hycoWSSender) CreateRelaySASToken() string {
	var uri = url.URL{Scheme: "http", Host: c.ns}
	escapedURI := url.QueryEscape(uri.String())
	fmt.Println("esapedURI: " + escapedURI)

	var unixSeconds = time.Now().Add(3600 * time.Second).Unix()
	var unixSecStr = fmt.Sprintf("%v", unixSeconds)
	fmt.Println("unixSeconds: " + unixSecStr)

	// The string-to-sign is a unique string constructed from the fields that must be verified in order to authorize the request.
	// The signature is an HMAC computed over the string-to-sign and key using the SHA256 algorithm, and then encoded using Base64 encoding.
	var stringToSign = escapedURI + "\n" + unixSecStr
	var signature = encrypt(c.key, stringToSign)

	token := "SharedAccessSignature sr=" + escapedURI + "&sig=" + url.QueryEscape(signature) + "&se=" + unixSecStr + "&skn=" + c.keyrule
	fmt.Println("token: " + token)

	return token
}

func encrypt(key string, stringToSign string) string {
	fmt.Println("key: ", key)
	fmt.Println("stingToSign: ", stringToSign)

	sig := hmac.New(sha256.New, []byte(key))
	sig.Write([]byte(stringToSign))
	sigBytes := sig.Sum(nil)

	return base64.StdEncoding.EncodeToString(sigBytes)
}
