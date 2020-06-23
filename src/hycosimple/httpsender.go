package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"errors"
)

type HycoClient interface {
	SendRequest() (*[]byte, error)
}

type hycoClient struct {
	url string
}

func (c hycoClient) SendRequest() (*[]byte, error) {
	fmt.Printf("Entering SendRequest ...")

	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", c.url, err.Error())
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Get on %s failed with status %s", c.url, resp.StatusCode)
		return nil, errors.New("unable to connect")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Unable to read response body %s", err.Error())
		return nil, err
	}

	fmt.Printf("Exit SendRequest.")
	return &body, nil
}
