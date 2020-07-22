package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

// HYCOListener is a simple listening client
type HYCOListener interface {
	// get listener uri
	GetRelayListenerURI(correlationID string) string

	// create sas token
	CreateRelaySASToken() string
}

// HycoListener defines relay options
type HycoListener struct {
	NS      string
	Path    string
	Keyrule string
	Key     string
}

// GetRelayListenerURI is a function to get listener uri
func (hyco HycoListener) GetRelayListenerURI(correlationID string) string {
	query := "sb-hc-action=listen"
	if correlationID != "" {
		query += "&sb-hc-id=" + correlationID
	}

	u := url.URL{Scheme: "wss", Host: hyco.NS + ":443", Path: "$hc/" + hyco.Path, RawQuery: query}
	fmt.Println(u.String())
	return u.String()
}

// CreateRelaySASToken is a function to create SAS token in listener
func (hyco HycoListener) CreateRelaySASToken() string {
	var uri = url.URL{Scheme: "http", Host: hyco.NS, Path: hyco.Path}
	escapedURI := url.QueryEscape(uri.String())
	fmt.Println("esapedURI: " + escapedURI)

	var unixSeconds = time.Now().Add(3600 * time.Second).Unix()
	var unixSecStr = fmt.Sprintf("%v", unixSeconds)
	fmt.Println("unixSeconds: " + unixSecStr)

	// The string-to-sign is a unique string constructed from the fields that must be verified in order to authorize the request.
	// The signature is an HMAC computed over the string-to-sign and key using the SHA256 algorithm, and then encoded using Base64 encoding.
	var stringToSign = escapedURI + "\n" + unixSecStr
	var signature = encrypt(hyco.Key, stringToSign)

	token := "SharedAccessSignature sr=" + escapedURI + "&sig=" + url.QueryEscape(signature) + "&se=" + unixSecStr + "&skn=" + hyco.Keyrule
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
