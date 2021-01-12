// This file is part of autosr.
//
// autosr is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// autosr is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with autosr.  If not, see <https://www.gnu.org/licenses/>.

package showroom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/bobbytrapz/autosr/options"
	"github.com/bobbytrapz/autosr/track"
	"github.com/gorilla/websocket"
)

// User in showroom
type User struct {
	Name string
	ID   int
}

// Event in showroom
type Event struct {
	At       time.Time
	bcsvrKey string
}

// Comment from chat
type Comment struct {
	User
	Event
	Text string
}

// Gift sent
type Gift struct {
	User
	Event
	ID     int
	Amount int
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	bcsvrHost = "online.showroom-live.com:443"
)

// channel for sending message to chat server
var msgSend chan []byte

type chatConnection struct {
	bcsvrKey  string // "59f5e9:vF3aKm5p"
	startedAt time.Time
}

func parseEvent(ev []byte) (object interface{}, err error) {
	// expecting = MSG	59eb86:iTW7lR5i	{created_at: 1547732326, u: 1707048, at: 6, t: 6}
	if !bytes.HasPrefix(ev, []byte("MSG")) {
		return
	}

	parts := bytes.Split(ev, []byte("\t"))
	bcsvrKey := string(parts[1])

	// look for start of json and unmarshal it
	ndx := bytes.Index(ev, []byte("{"))
	if ndx == -1 {
		return
	}
	j := ev[ndx:]
	var data map[string]interface{}
	if err = json.Unmarshal(j, &data); err != nil {
		err = fmt.Errorf("showroom.parseEvent: %s", err)
		return
	}

	switch data["t"] {
	case "1":
		// comment
		return Comment{
			User: User{
				Name: data["ac"].(string),
				ID:   int(data["u"].(float64)),
			},
			Event: Event{
				bcsvrKey: bcsvrKey,
				At:       time.Unix(int64(data["created_at"].(float64)), 0),
			},
			Text: data["cm"].(string),
		}, nil
	case "2":
		// gift
		return Gift{
			User: User{
				Name: data["ac"].(string),
				ID:   int(data["u"].(float64)),
			},
			Event: Event{
				bcsvrKey: bcsvrKey,
				At:       time.Unix(int64(data["created_at"].(float64)), 0),
			},
			ID:     int(data["u"].(float64)),
			Amount: int(data["n"].(float64)),
		}, nil
	case 8:
		// telop change
		return
	case 6:
		// ?
		return
	default:
		return
	}
}

// RecordEvent stores the event in the database
func RecordEvent(logFile *os.File, ev []byte) {
	o, err := parseEvent(ev)
	if err != nil {
		log.Printf("showroom.RecordEvent: parseEvent: %s", err)
		return
	}
	switch v := o.(type) {
	// could write comments to recorded/[name]/[date].log
	// now use tail -F [date].log to follow all the chat
	case Comment:
		fmt.Printf("record comment: %v (%v)\n", v, logFile)
	case Gift:
		fmt.Printf("record gift: %v (%v)\n", v, logFile)
	case nil:
	default:
	}
}

// WatchEvents tracks websockets events for a given key
// a separate websocket connection is made for each chat room
func WatchEvents(ctx context.Context, bcsvrKey string) {
	url := url.URL{
		Scheme: "wss",
		Host:   bcsvrHost,
	}
	ua := options.Get("user_agent")

	// dial
	dialer := websocket.Dialer{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  5 * time.Second,
		EnableCompression: true,
	}
	header := http.Header{
		"User-Agent": []string{ua},
	}
	log.Printf("showroom.connectChatServer: dial %s (%v)", url.String(), header)
	c, _, err := dialer.DialContext(ctx, url.String(), header)
	if err != nil {
		panic(err)
	}
	log.Printf("showroom.connectChatServer: connected")

	// subscribe to chat
	subcmd := []byte(fmt.Sprintf("SUB\t%s", bcsvrKey))
	log.Printf("showroom.connectChatServer: %s", subcmd)
	if err := c.WriteMessage(websocket.TextMessage, subcmd); err != nil {
		log.Printf("showroom.connectChatServer: tried to send: %s", err)
		return
	}

	// open chat log
	logFile, err := os.OpenFile("chat.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("showroom.SubscribeChat: failed to open log file: %s", err)
		return
	}

	// fmt.Fprintf(logFile, "# Started at %s", time.Now())

	// commands
	pingcmd := []byte("PING\tshowroom")
	// quitcmd := []byte("QUIT")

	msgSend = make(chan []byte, 1)
	done := make(chan struct{}, 1)
	// read
	track.Add(1)
	go func() {
		defer track.Done()
		defer close(done)
		defer logFile.Close()
		log.Printf("showroom.connectChatServer: read")
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("showroom.connectChatServer: err:", err)
				return
			}
			RecordEvent(logFile, msg)
		}
	}()

	// write
	track.Add(1)
	go func() {
		defer track.Done()

		log.Printf("showroom.connectChatServer: write")
		defer c.Close()

		pingTicker := time.NewTicker(1 * time.Minute)
		defer pingTicker.Stop()
		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
				log.Printf("showroom.connectChatServer: %s", pingcmd)
				if err := c.WriteMessage(websocket.TextMessage, pingcmd); err != nil {
					log.Println("showroom.connectChatServer: tried to send ping:", err)
				}
			case <-ctx.Done():
				/*
					// sending quit causes abnormal closure
					log.Printf("showroom.connectChatServer: %s", quitcmd)
					if err := c.WriteMessage(websocket.TextMessage, quitcmd); err != nil {
						log.Println("showroom.connectChatServer: tried to send quit:", err)
					}
				*/

				log.Println("showroom.connectChatServer: close...")
				if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
					log.Println("showroom.connectChatServer: close:", err)
					return
				}

				// wait a bit for it to close
				select {
				case <-done:
				case <-time.After(1 * time.Second):
				}

				// stop write
				return
			}
		}
	}()

	return
}
