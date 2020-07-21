package main

import (
	"fmt"
	"io"
	"net/http"
)

// HYCOListener is a simple listening client
type HYCOListener interface {
	createRelayListsenURI() string
	Start()
}

type hycoListener struct {
	// Contains necessary listener variables
	ns      string
	path    string
	keyrule string
	key     string
}

func (listenerClient hycoListener) createRelayListsenURI() string {
	var uri = "https://" + listenerClient.ns + "/" + listenerClient.path
	return uri
}

func (listenerClient hycoListener) Start() {
	fmt.Printf("Entering Start...\n")
	fmt.Printf("Now listening on %s : %s \n", listenerClient.ns, listenerClient.path)
	http.HandleFunc("/", httpReqHandler)
	err := http.ListenAndServe(listenerClient.ns+":"+listenerClient.path, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("Exit Start. \n")
}

func httpReqHandler(w http.ResponseWriter, r *http.Request) {
	var responseContent = fmt.Sprintf("Received: %s on %s with query %s", r.Method, r.URL.Path, r.URL.RawQuery)
	fmt.Printf(responseContent)
	defer r.Body.Close()

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, responseContent)

	return
}

func main() {
	var listenClient HYCOListener
	listenClient = hycoListener{ns: "gorelay.servicebus.windows.net", path: "noclientauth", keyrule: "managepolicy", key: "GYx32+NyDOXroUaDpflfhlAz/FeioiRsV6IqCb5oDZs="}
	
	// Accept connections in :8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Get on %s failed. Details: %s", listenClient.createRelayListsenURI(), err)
	} else {
		fmt.Printf("server is listening")
	}
}
