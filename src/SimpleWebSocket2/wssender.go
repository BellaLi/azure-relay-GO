// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"log"
)

func main() {
	log.SetFlags(0)

	var client HYCOSender
	client = hycoSender{
		ns:                 "gorelay.servicebus.windows.net",
		path:               "yesclientauth",
		keyrule:            "managepolicy",
		key:                "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE=",
		clientAuthRequired: true}

	sasToken := client.CreateRelaySASToken()
	client.ConnectRelayWS(sasToken)
}
