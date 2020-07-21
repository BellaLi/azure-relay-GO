package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HYCOSender is a simple sending client
type HYCOSender interface {
	GetRelayHTTPSURI(correlationID string) string
	CreateRelaySASToken() string
	SendRequest(method, body, sasToken string) (*[]byte, error)
}

type hycoSender struct {
	ns      string
	path    string
	keyrule string
	key     string
}

func main() {
	var client HYCOSender
	client = hycoSender{
		ns:      "gorelay.servicebus.windows.net",
		path:    "yesclientauth",
		keyrule: "managepolicy",
		key:     "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE="}

	sasToken := client.CreateRelaySASToken()
	uri := client.GetRelayHTTPSURI("")
	// try GET
	resp, err := client.SendRequest("GET", "", sasToken)
	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", uri, err.Error())
	} else {
		fmt.Printf("%s", resp)
	}

	// try POST
	resp, err = client.SendRequest("POST", "Hey Jude!", sasToken)
	if err != nil {
		fmt.Printf("POST on %s failed. Details: %s", uri, err.Error())
	} else {
		fmt.Printf("%s", resp)
	}
}

func (c hycoSender) GetRelayHTTPSURI(correlationID string) string {
	var query string
	if correlationID != "" {
		query = "sb-hc-id=" + correlationID
	}

	u := url.URL{Scheme: "https", Host: c.ns, Path: c.path, RawQuery: query}
	fmt.Println(u.String())
	return u.String()
}

func (c hycoSender) SendRequest(method, body, sasToken string) (*[]byte, error) {
	fmt.Printf("Entering SendRequest ... \n")
	uri := c.GetRelayHTTPSURI("")

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
		sasToken = c.CreateRelaySASToken()
	}
	req.Header.Add("ServiceBusAuthorization", sasToken)
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

	fmt.Printf("Exit SendRequest. \n")
	return &respBody, nil
}

func (c hycoSender) CreateRelaySASToken() string {
	var uri = c.GetRelayHTTPSURI("")
	uri = strings.Replace(uri, "https", "http", 1)
	escapedURI := url.QueryEscape(uri)
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
