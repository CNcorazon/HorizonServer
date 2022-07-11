package request

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"server/model"
	"time"

	"github.com/gorilla/websocket"
)

func WSRequest(wsurl string, route string) *websocket.Conn {
	rand.Seed(time.Now().UnixNano())
	URL := wsurl + route
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial(URL, nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	return conn
}

func ClientRegister(httpurl string, route string, clientdata model.ClientForwardRequest) model.ClientForwardResponse {
	URL := httpurl + route
	data := clientdata
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}
	request, _ := http.NewRequest("POST", URL, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	body, _ := ioutil.ReadAll(response.Body)
	var res model.ClientForwardResponse
	json.Unmarshal(body, &res)
	return res
}

// func SynTxpool(httpurl string, route string)
