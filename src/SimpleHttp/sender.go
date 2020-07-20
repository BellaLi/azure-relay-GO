package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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

func (c hycoSender) SendRequest() (*[]byte, error) {
	fmt.Printf("Entering SendRequest ... \n")
	uri := c.GetRelayHTTPSURI()
	req, err := http.NewRequest("GET", uri, nil)
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
	client = hycoSender{ns: "gorelay.servicebus.windows.net", path: "noclientauth", keyrule: "managepolicy", key: "GYx32+NyDOXroUaDpflfhlAz/FeioiRsV6IqCb5oDZs="}
	resp, err := client.SendRequest()

	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", client.GetRelayHTTPSURI(), err.Error())
	} else {
		fmt.Printf("%s", resp)
	}
}
