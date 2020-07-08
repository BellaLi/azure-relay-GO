package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println(len(os.Args), os.Args)
	var args = os.Args
	var arg1 = args[1]
	var arg2 = args[2]

	if arg1 == "http" && arg2 == "sender" {
		targeturl := "http://localhost:8080"

		fmt.Printf("Access %s \n", targeturl)

		var client HttpSender
		client = httpSender{targeturl}
		resp, err := client.SendRequest()

		if err != nil {
			fmt.Printf("Get on %s failed. Details: %s", targeturl, err.Error())
		} else {
			fmt.Printf("%s", resp)
		}
	} else if arg1 == "http" && arg2 == "listener" {
		var server HttpListener
		server = httpListener{"localhost", 8080}
		server.Start()
	} else {
		fmt.Printf("Cannot recognize '%s' and '%s'", arg1, arg2)
	}

}
