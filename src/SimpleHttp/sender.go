package main

import (
	"fmt"
)

func main() {
	var client HYCOSender
	client = hycoSender{
		ns:                 "gorelay.servicebus.windows.net",
		path:               "yesclientauth", //"noclientauth",
		keyrule:            "managepolicy",
		key:                "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE=", // "GYx32+NyDOXroUaDpflfhlAz/FeioiRsV6IqCb5oDZs=", //
		clientAuthRequired: true}

	sasToken := client.CreateRelaySASToken()
	uri := client.GetRelayHTTPSURI("")

	// try http GET
	resp, err := client.SendRequest("GET", "", sasToken)
	if err != nil {
		fmt.Printf("Get on %s failed. Details: %s", uri, err.Error())
	} else {
		fmt.Printf("%s", resp)
	}

	// try http POST
	resp, err = client.SendRequest("POST", "Hey Jude!", sasToken)
	if err != nil {
		fmt.Printf("POST on %s failed. Details: %s", uri, err.Error())
	} else {
		fmt.Printf("%s", resp)
	}
}
