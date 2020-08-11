package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type acceptInner struct {
	ID             string         `json:"id"`
	Address        string         `json:"address"`
	ConnectHeaders connectHeaders `json:"connectHeaders"`
	RemoteEndpoint remoteEndpoint `json:"remoteEndpoint"`
}

// {"Sec-WebSocket-Key":"NsoFiXBuR3i2nE8Tx0+maA==","Sec-WebSocket-Version":"13","Connection":"Upgrade","Upgrade":"websocket",
// "Host":"gorelay.servicebus.windows.net:443","User-Agent":"Go-http-client\/1.1"}
type connectHeaders struct {
	SecWebSocketKey     string `json:"Sec-WebSocket-Key"`
	SecWebSocketVersion string `json:"Sec-WebSocket-Version"`
	Connection          string `json:"Connection"`
	Upgrade             string `json:"Upgrade"`
	Host                string `json:"Host"`
	UserAgent           string `json:"User-Agent"`
}

type remoteEndpoint struct {
	Address string `json:"address"`
	Port    int32  `json:"port"`
}

type requestInner struct {
	ID            string `json:"id"`
	Address       string `json:"address"`
	Method        string `json:"method"`
	RequestTarget string `json:"requestTarget"`
	Body          bool   `json:"body"`
}

type outer struct {
	Request requestInner
	Accept  acceptInner
}

type respEvent struct {
	MessageType int
	respData    string
}

var relay HycoListener
var wsConnections map[string]*websocket.Conn

func startListener(ctx context.Context) {
	c, hcID, _, err := relayConnect(ctx, nil)
	if err != nil {
		fmt.Printf("Unable to connect to relay. %s \n", err.Error())
	}
	fmt.Printf("Connected to %s/%s \n", relay.NS, relay.Path)

	err = recieveMessages(ctx, c, hcID)
	fmt.Printf("recieveMessages Error: %s", err.Error())
}

func acceptClient(ctx context.Context, acceptMsg *acceptInner) {
	if wsConnections[acceptMsg.ID] != nil {
		fmt.Printf("Error: Connection to %s already exists! \n", acceptMsg.ID)
		return
	}

	c, hcID, _, err := relayConnect(ctx, acceptMsg)
	if err != nil {
		fmt.Printf("[%s] Unable to accept: %s \n", acceptMsg.ID, err.Error())
		return
	}

	wsConnections[acceptMsg.ID] = c
	fmt.Printf("[%s] Connected. \n", acceptMsg.ID)

	err = recieveMessages(ctx, c, hcID)
	fmt.Printf("[%s] recieveMessages Error: %s \n", acceptMsg.ID, err.Error())
}

func relayConnect(ctx context.Context, acceptMsg *acceptInner) (con *websocket.Conn, hcID string, httpStatus int, err error) {
	var httpResp *http.Response
	httpStatus = -1
	var u string

	headers := make(http.Header)

	if acceptMsg == nil {
		hcID = uuid.New().String()
		u = relay.GetRelayListenerURI(hcID)

		sbaHeaderName := "ServiceBusAuthorization"
		sbaHeaderValue := relay.CreateRelaySASToken()
		headers[sbaHeaderName] = []string{sbaHeaderValue}
	} else {
		u = acceptMsg.Address
		// headers["sec-webSocket-key"] = []string{acceptMsg.ConnectHeaders.SecWebSocketKey}
		// headers["sec-websocket-version"] = []string{acceptMsg.ConnectHeaders.SecWebSocketVersion}
		// headers["host"] = []string{acceptMsg.ConnectHeaders.Host}
		// headers["upgrade"] = []string{acceptMsg.ConnectHeaders.Upgrade}
		// headers["connection"] = []string{acceptMsg.ConnectHeaders.Connection}
		// headers["user-agent"] = []string{acceptMsg.ConnectHeaders.UserAgent}

		re, _ := json.Marshal(acceptMsg.RemoteEndpoint)
		headers["remoteEndpoint"] = []string{string(re)}
	}
	fmt.Println(u)

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
	respQ := make(chan respEvent, 5)
	defer close(respQ)
	go func() {
		fmt.Println("Pinging relay every 15 seconds")
		ticker := time.NewTicker(15 * time.Second)
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
					fmt.Println("respQ ok is false, breaking the loop.")
					break rrloop
				}
				err := c.WriteMessage(resp.MessageType, []byte(resp.respData))
				if err != nil {
					fmt.Println("Failed to write to ws." + err.Error())
					c.Close()
					break rrloop
				}
				fmt.Printf("response message sent. %d: %s \n", resp.MessageType, resp.respData)
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
			case <-time.After(time.Minute * 59):
			case <-rwCtx.Done():
				return
			}

			fmt.Println("Renewing relay token")
			newToken := relay.CreateRelaySASToken()

			fmt.Println("Renewed relay token")
			payload := `{"renewToken":{"token":"` + newToken + `"}}`
			respQ <- respEvent{websocket.TextMessage, payload}
		}
	}()

	for {
		var message []byte
		var mt int
		var err error

		mt, message, err = c.ReadMessage()
		fmt.Printf("Recv Type: %d. Message: %s", mt, message[:])
		if err != nil {
			return errors.New("Error while reading header message on ws con#:" + err.Error())
		}

		// - ignore pong messages
		if mt == websocket.PongMessage {
			fmt.Println("Pong message received")
			continue
		}

		if mt != websocket.TextMessage {
			return errors.New("Header message is not of expected type (text)")
		}

		// todo: parse websocket message
		var header outer
		err = json.Unmarshal(message, &header)
		if err != nil {
			return errors.New("Unable to decode request header. " + err.Error())
		}

		var requestID string

		if header.Accept.ID != "" {
			/* websocket sample
			{"accept":{
				"address":"wss:\/\/g17-prod-by3-010-sb.servicebus.windows.net\/$hc\/yesclientauth?sb-hc-action=accept&sb-hc-id=ca496b91-f5a3-4761-8eda-9a66dd9a2558_G17_G30",
				"id":"ca496b91-f5a3-4761-8eda-9a66dd9a2558_G17_G30",
				"connectHeaders":{"Sec-WebSocket-Key":"NsoFiXBuR3i2nE8Tx0+maA==","Sec-WebSocket-Version":"13","Connection":"Upgrade","Upgrade":"websocket","Host":"gorelay.servicebus.windows.net:443","User-Agent":"Go-http-client\/1.1"},
				"remoteEndpoint":{"address":"73.83.210.109","port":62917}
			}}
			*/

			// handle Accept
			go func() {
				acceptCtx := context.Background()
				acceptClient(acceptCtx, &header.Accept)
			}()

			continue
		}

		if header.Request.ID == "" {
			/* http sample
			{"request":{"address":"wss://g12-prod-by3-010-sb.servicebus.windows.net/$hc/yesclientauth?sb-hc-action=request&sb-hc-id=c126fddd-5ca6-430f-9b10-e2188d1ed0d4_G12",
			"id":"c126fddd-5ca6-430f-9b10-e2188d1ed0d4_G12","requestTarget":"/yesclientauth","method":"POST","remoteEndpoint":{"address":"73.83.210.109","port":62915},
			"requestHeaders":{"Content-Type":"application/json; charset=utf-8","Accept-Encoding":"gzip","Host":"gorelay.servicebus.windows.net","User-Agent":"Go-http-client/1.1","Via":"1.1 gorelay.servicebus.windows.net"},"body":true}}
			*/
			return errors.New("Cannot find request Id in incoming payload: " + string(message))
		}

		requestID = header.Request.ID

		var resp = `{"response":{"requestId":"` + requestID + `","statusCode":"200","responseHeaders":{"Content-Type":"application/json; charset=utf-8"},"body":true}}`
		respQ <- respEvent{websocket.TextMessage, resp}

		if header.Request.Body {
			mt, message, err = c.ReadMessage()
			fmt.Printf("Recv Type: %d. Message: %s", mt, message[:])

			if err != nil {
				return errors.New("Error while reading request body on ws con#:" + hcID + ". " + err.Error())
			}
		} else {
			message = []byte("noBody")
		}
		httpReqHandler(&header.Request, message, respQ)
	}
}

func httpReqHandler(r *requestInner, body []byte, w chan respEvent) {
	var responseContent = fmt.Sprintf("Received: %s on %s with ID %s and body %s", r.Method, r.RequestTarget, r.ID, body)
	resp := `{"echo":"` + responseContent + `"}`
	fmt.Printf(resp)

	w <- respEvent{websocket.BinaryMessage, resp}
	return
}

func main() {
	relay = HycoListener{
		NS:      "gorelay.servicebus.windows.net",
		Path:    "yesclientauth",
		Keyrule: "managepolicy",
		Key:     "SkJUQP/1FTjT/Z0QcXwgUnqRUCnSimo9HORcyTxVtgE="}
	wsConnections = make(map[string]*websocket.Conn)
	fmt.Println("Starting...")

	ctx := context.Background()
	startListener(ctx)
}
