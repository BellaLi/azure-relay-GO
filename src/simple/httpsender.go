package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// HttpSender is a simple client
type HttpSender interface {
	SendRequest() (*[]byte, error)
}

type httpSender struct {
	url string
}

func (c httpSender) SendRequest() (*[]byte, error) {
	fmt.Printf("Entering SendRequest ... \n")

	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s \n", c.url, err.Error())
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Get on %s failed with status %d \n", c.url, resp.StatusCode)
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
