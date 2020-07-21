package main

import (
	"context"
	"net/http"
	"net/url"
	"errors"
	"os"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var relay relayConfig

type relayConfig struct {
	Namespace string `json:"namespace"`
	HC    string `json:"hc"`
	Key string `json:"key"`
}

func startListener(ctx context.Context) {
	c, hcId, _, err := relayConnect(ctx)
	if err != nil {
		fmt.Println("Unable to connect to relay. %s", err.Error())
	}
	fmt.Println("Connected to %s/%s", relay.Namespace, relay.HC)
	err = recieveMessages(ctx, c, hcId)
}

func relayConnect(ctx context.Context) (con *websocket.Conn, hcId string, httpStatus int, err error) {

	var httpResp *http.Response
	httpStatus = -1

	relayNS := relay.Namespace
	hcname := relay.HC

	hcId = uuid.New().String()
	u := url.URL{Scheme: "wss", Host: relayNS + ".servicebus.windows.net", Path: "/$hc/" + hcname + "?sb-hc-action=listen&sb-hc-id=" + hcId}

	headers := make(http.Header)
	sbaHeaderName := "ServiceBusAuthorization"
	sbaHeaderValue := relay.Key
	headers[sbaHeaderName] = []string{sbaHeaderValue}

	con, httpResp, err = websocket.DefaultDialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		errStr := ""
		if httpResp != nil {
			errStr += httpResp.Status + ". "
		}
		err = errors.New(errStr + err.Error())
	}

	if httpResp != nil {
		httpStatus = httpResp.StatusCode
	}
	return
}

/* connect to relay endpoint, listen and send back the messages */
func recieveMessages(ctx context.Context, c *websocket.Conn, hcId string) error {
	//var err error	
	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		fmt.Printf("recv: %s", message)
		resp := "Echo: " //+ message
		if err != nil {
			return errors.New("Error while reading request body on ws con#:" + hcId + ". " + err.Error())
		}
		err = c.WriteMessage(websocket.TextMessage, []byte(resp))

	}
}


func main() {
	if(len(os.Args)<4){
		fmt.Println("receiver.exe [ns] [hc] [key]")
		exit(1)
	}
	relay.Namespace = os.Args[1]
	relay.HC = os.Args[2]
	relay.Key = os.Args[3]
	
	ctx := context.Background()
	go startListener(ctx)

}
