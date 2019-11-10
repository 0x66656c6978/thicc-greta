package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/0x66656c6978/thiccgreta/indexer"
	"github.com/0x66656c6978/thiccgreta/websocket"
)

func serveHTTP(addr string, hub *websocket.Hub) {
	httpServiceLogger := log.Logger{}
	httpServiceLogger.SetPrefix("http.service: ")
	httpServiceLogger.SetOutput(log.Writer())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		websocket.Serve(hub, w, r)
	})
	httpServiceLogger.Printf("Listening on %v", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		httpServiceLogger.Fatal(err)
	}
}

// receive offer messages from the offer channel, serialize them to json
// and then broadcast them to connected websocket clients
func broadcastOffers(hub *websocket.Hub, offerChannel chan indexer.Offer) {
	for {
		offer := <-offerChannel
		jsonOffer, jsonErr := json.Marshal(offer)
		if jsonErr != nil {
			panic(jsonErr)
		}
		hub.Broadcast(jsonOffer)
	}
}

func main() {
	offerChannel := make(chan indexer.Offer)
	league := flag.String("league", "Standard", "The league to stream items from")
	addr := flag.String("addr", ":8080", "http service address")
	flag.Parse()

	hub := websocket.NewHub()
	go hub.Run()
	go serveHTTP(*addr, hub)
	go broadcastOffers(hub, offerChannel)

	// the indexer runs on the main routine
	indexer.Run(*league, offerChannel)
}
