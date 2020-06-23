package main

import (
	"fmt"
)

func main() {
	targeturl := "http://example.com"
	
	fmt.Printf("Access %s \n", targeturl)

	var client HycoClient = hycoClient{targeturl}
	resp, err := client.SendRequest()

	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", err.Error())
	} else {
		fmt.Printf("%s", resp)
	}
}