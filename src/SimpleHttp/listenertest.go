package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var relay HycoListener

func httpReqHandler2(w http.ResponseWriter, r *http.Request) {
	var responseContent = fmt.Sprintf("Received: %s on %s with query %s", r.Method, r.URL.Path, r.URL.RawQuery)
	fmt.Printf(responseContent)
	defer r.Body.Close()

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, responseContent)

	return
}

func startListener(ctx context.Context) {
	c, hcID, _, err := relayConnect(ctx)
	if err != nil {
		fmt.Println("Unable to connect to relay. %s", err.Error())
	}
	fmt.Println("Connected to %s/%s", relay.NS, relay.Path)
	err = recieveMessages(ctx, c, hcID)
}

func relayConnect(ctx context.Context) (con *websocket.Conn, hcID string, httpStatus int, err error) {
	var httpResp *http.Response
	httpStatus = -1

	hcID = uuid.New().String()
	u := relay.GetRelayListenerURI(hcID)

	headers := make(http.Header)
	sbaHeaderName := "ServiceBusAuthorization"
	sbaHeaderValue := relay.CreateRelaySASToken()
	headers[sbaHeaderName] = []string{sbaHeaderValue}

	con, httpResp, err = websocket.DefaultDialer.DialContext(ctx, u, headers)
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
func recieveMessages(ctx context.Context, c *websocket.Conn, hcID string) error {
	defer c.Close()

	/* setup response worker */
	respQ := make(chan string, 5)
	defer close(respQ)
	go func() {
		fmt.Println("Pinging relay every 2 seconds")
		ticker := time.NewTicker(2 * time.Second)
	rrloop:
		for {
			select {
			case <-ticker.C:
				fmt.Println("Sending ping message")
				err := c.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					fmt.Println("Failed to send ping message on ws." + err.Error())
					c.Close()
					break rrloop
				}

			case resp, ok := <-respQ:
				fmt.Println("Sending response message.")
				if ok == false {
					break rrloop
				}
				err := c.WriteMessage(websocket.TextMessage, []byte(resp))
				fmt.Println("response message sent:" + resp)
				if err != nil {
					fmt.Println("Failed to write to ws. " + err.Error())
					c.Close()
					break rrloop
				}
			}
		}
		ticker.Stop()
		fmt.Println("Exiting response worker")
		return
	}()

	/* setup renewing worker */
	rwCtx, rwCancel := context.WithCancel(ctx)
	var rwWG sync.WaitGroup
	rwWG.Add(1)
	defer func() {
		rwCancel()
		rwWG.Wait()
	}()
	go func() {
		defer func() {
			fmt.Println("Exiting renewing worker")
			rwWG.Done()
		}()

		for {
			select {
			case <-time.After(time.Minute * 1):
			case <-rwCtx.Done():
				return
			}

			fmt.Println("Renewing relay token")
			newToken := relay.CreateRelaySASToken()

			fmt.Println("Renewed relay token")
			payload := `{"renewToken":{"token":"` + newToken + `"}}`
			respQ <- payload
		}
	}()

	for {
		type inner struct {
			Id      string
			Address string
		}
		type outer struct {
			Request inner
			Accept  inner
		}
		var header outer

		mt, message, err := c.ReadMessage()
		err = json.Unmarshal(message, &header)
		if err != nil {
			return errors.New("Unable to decode request header. " + err.Error())
		}

		/*
			{"request":{"address":"wss://g12-prod-by3-010-sb.servicebus.windows.net/$hc/yesclientauth?sb-hc-action=request&sb-hc-id=c126fddd-5ca6-430f-9b10-e2188d1ed0d4_G12",
			"id":"c126fddd-5ca6-430f-9b10-e2188d1ed0d4_G12","requestTarget":"/yesclientauth","method":"POST","remoteEndpoint":{"address":"73.83.210.109","port":62915},
			"requestHeaders":{"Content-Type":"application/json; charset=utf-8","Accept-Encoding":"gzip","Host":"gorelay.servicebus.windows.net","User-Agent":"Go-http-client/1.1","Via":"1.1 gorelay.servicebus.windows.net"},"body":true}}
		*/
		requestId := header.Request.Id

		if requestId == "" {
			/*
				{"accept":{"address":"wss:\/\/g17-prod-by3-010-sb.servicebus.windows.net\/$hc\/yesclientauth?sb-hc-action=accept&sb-hc-id=ca496b91-f5a3-4761-8eda-9a66dd9a2558_G17_G30",
				"id":"ca496b91-f5a3-4761-8eda-9a66dd9a2558_G17_G30","connectHeaders":{"Sec-WebSocket-Key":"NsoFiXBuR3i2nE8Tx0+maA==","Sec-WebSocket-Version":"13","Connection":"Upgrade",
				"Upgrade":"websocket","Host":"gorelay.servicebus.windows.net:443","User-Agent":"Go-http-client\/1.1"},"remoteEndpoint":{"address":"73.83.210.109","port":62917}}}
			*/
			requestId = header.Accept.Id
		}

		if err != nil {
			fmt.Println("read:", err)
			return nil
		}

		if mt == websocket.TextMessage {
			fmt.Println("Recv TextMessage: " + string(message[:]))
		} else {
			fmt.Printf("Recv Type: %d. Message: %s", mt, string(message[:]))
		}

		// todo: this response doesn't work well. need to figure out what to response for Accept message & for http requests
		var resp = `{"response":{"requestId":"` + requestId + `","statusCode":"200","responseHeaders":{"Content-Type":"application/json; charset=utf-8"},"body":false}}`
		respQ <- resp
	}
}

func main() {
	relay = HycoListener{
		NS:      "gorelay.servicebus.windows.net",
		Path:    "yesclientauth",
		Keyrule: "managepolicy",
		Key:     "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE="}

	fmt.Println("Starting...")

	ctx := context.Background()
	startListener(ctx)
}
