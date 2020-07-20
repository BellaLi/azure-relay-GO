package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HYCOSender is a simple sending client
type HYCOSender interface {
	GetRelayHTTPSURI() string
	SendRequest() (*[]byte, error)
}

type hycoSender struct {
	ns      string
	path    string
	keyrule string
	key     string
}

func (c hycoSender) GetRelayHTTPSURI() string {
	var uri = "https://" + c.ns + "/" + c.path
	return uri
}

func (c hycoSender) createRelayToken() string {
	var uri = c.GetRelayHTTPSURI()
	uri = strings.Replace(uri, "https", "http", 1)
	var unixSeconds = time.Now().Add(3600 * time.Second).Unix()

	escapedURI := url.QueryEscape(uri)

	// The string-to-sign is a unique string constructed from the fields that must be verified in order to authorize the request.
	// The signature is an HMAC computed over the string-to-sign and key using the SHA256 algorithm, and then encoded using Base64 encoding.
	var stringToSign = url.QueryEscape(uri) + "\n" + fmt.Sprintf("%v", unixSeconds)
	var signature = encrypt(c.key, stringToSign)

	token := "SharedAccessSignature sr=" + escapedURI + "&sig=" + url.QueryEscape(signature) + "&se=" + fmt.Sprintf("%v", unixSeconds) + "&skn=" + c.keyrule
	fmt.Println("token: " + token)
	fmt.Println("esapedURI: " + escapedURI)

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

func (c hycoSender) SendRequest() (*[]byte, error) {
	fmt.Printf("Entering SendRequest ... \n")
	uri := c.GetRelayHTTPSURI()
	req, err := http.NewRequest("GET", uri, nil)
	req.Header.Add("ServiceBusAuthorization", c.createRelayToken())

	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Unable to read response body %s \n", err.Error())
		return nil, err
	}

	fmt.Printf("Exit SendRequest. \n")
	return &body, nil
}

func main() {
	var client HYCOSender
	client = hycoSender{ns: "gorelay.servicebus.windows.net", path: "yesclientauth", keyrule: "managepolicy", key: "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE="}
	resp, err := client.SendRequest()

	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", client.GetRelayHTTPSURI(), err.Error())
	} else {
		fmt.Printf("%s", resp)
	}
}
