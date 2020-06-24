package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// HycoServer is a simple listener
type HycoServer interface {
	Start()
}

type hycoServer struct {
	Address string
	Port    int
}

func (server hycoServer) Start() {
	fmt.Printf("Entering Start...\n")
	fmt.Printf("Now listening on %s:%d \n", server.Address, server.Port)
	http.HandleFunc("/", httpReqHandler)
	err := http.ListenAndServe(server.Address+":"+strconv.Itoa(server.Port), nil)
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
