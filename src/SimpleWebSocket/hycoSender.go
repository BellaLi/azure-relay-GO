package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// HYCOSender is a simple sending client
type HYCOSender interface {
	GetRelayHTTPSURI(correlationID string) string
	GetRelayWSURI(correlationID string) string
	CreateRelaySASToken() string
	SendRequest(method, body, sasToken string) (*[]byte, error)
	ConnectRelayWS(sasToken string)
}

type hycoSender struct {
	ns                 string
	path               string
	keyrule            string
	key                string
	clientAuthRequired bool
}

func (hyco hycoSender) GetRelayHTTPSURI(correlationID string) string {
	var query string
	if correlationID != "" {
		query = "sb-hc-id=" + correlationID
	}

	u := url.URL{Scheme: "https", Host: hyco.ns, Path: hyco.path, RawQuery: query}
	fmt.Println(u.String())
	return u.String()
}

func (hyco hycoSender) GetRelayWSURI(correlationID string) string {
	query := "sb-hc-action=connect"
	if correlationID != "" {
		query = "&sb-hc-id=" + correlationID
	}

	u := url.URL{Scheme: "wss", Host: hyco.ns + ":443", Path: "$hc/" + hyco.path, RawQuery: query}
	fmt.Println(u.String())
	return u.String()
}

func (hyco hycoSender) SendRequest(method, body, sasToken string) (*[]byte, error) {
	fmt.Println("Entering SendRequest ...")
	uri := hyco.GetRelayHTTPSURI("")

	var bodyIO io.Reader
	if body == "" {
		bodyIO = nil
	} else {
		bodyIO = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, uri, bodyIO)
	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	if sasToken == "" {
		sasToken = hyco.CreateRelaySASToken()
	}

	if hyco.clientAuthRequired {
		req.Header.Add("ServiceBusAuthorization", sasToken)
	}
	req.Header.Add("content-type", "application/json; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s \n", uri, err.Error())
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Get on %s failed with status %d \n", uri, resp.StatusCode)
		return nil, errors.New("unable to connect")
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Unable to read response body %s \n", err.Error())
		return nil, err
	}

	fmt.Println("Exit SendRequest.")
	return &respBody, nil
}

func (hyco hycoSender) ConnectRelayWS(sasToken string) {
	fmt.Println("Entering ConnectRelayWS ...")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := hyco.GetRelayWSURI("")
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

func (hyco hycoSender) CreateRelaySASToken() string {
	var uri = url.URL{Scheme: "http", Host: hyco.ns, Path: hyco.path}
	escapedURI := url.QueryEscape(uri.String())
	fmt.Println("esapedURI: " + escapedURI)

	var unixSeconds = time.Now().Add(3600 * time.Second).Unix()
	var unixSecStr = fmt.Sprintf("%v", unixSeconds)
	fmt.Println("unixSeconds: " + unixSecStr)

	// The string-to-sign is a unique string constructed from the fields that must be verified in order to authorize the request.
	// The signature is an HMAC computed over the string-to-sign and key using the SHA256 algorithm, and then encoded using Base64 encoding.
	var stringToSign = escapedURI + "\n" + unixSecStr
	var signature = encrypt(hyco.key, stringToSign)

	token := "SharedAccessSignature sr=" + escapedURI + "&sig=" + url.QueryEscape(signature) + "&se=" + unixSecStr + "&skn=" + hyco.keyrule
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
