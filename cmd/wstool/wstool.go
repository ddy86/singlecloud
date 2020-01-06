package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

var (
	path = flag.String("path", "", "singlecloud websocket path")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	openLogWs()
}

func openLogWs() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(*path)
	if err != nil {
		log.Fatal("url parse failed:", err)
	}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	defer c.Close()

	done := make(chan struct{})
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				close(done)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	select {
	case <-done:
		return
	case <-interrupt:
		log.Println("interrupt")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return
	}
}
