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
	"time"

	"github.com/bobbytrapz/autosr/options"
	"github.com/bobbytrapz/autosr/track"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	bcsvrHost = "online.showroom-live.com:443"
	bcsvrKey  = "59f5e9:vF3aKm5p"
)

func parseEvent(ev []byte) (object interface{}, err error) {
	// expecting = MSG	59eb86:iTW7lR5i	{created_at: 1547732326, u: 1707048, at: 6, t: 6}
	if !bytes.HasPrefix(ev, []byte("MSG")) {
		return
	}

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
			Text: data["cm"].(string),
			At:   time.Unix(int64(data["created_at"].(float64)), 0),
		}, nil
	case "2":
		// gift
		return Gift{
			User: User{
				Name: data["ac"].(string),
				ID:   int(data["u"].(float64)),
			},
			ID:     int(data["u"].(float64)),
			Amount: int(data["n"].(float64)),
			At:     time.Unix(int64(data["created_at"].(float64)), 0),
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
func RecordEvent(ev []byte) {
	o, err := parseEvent(ev)
	if err != nil {
		log.Printf("[showroom] RecordEvent: parseEvent: %s", err)
		return
	}
	switch v := o.(type) {
	// could write comments to recorded/[name]/[date].log
	// now use tail -F [date].log to follow all the chat
	case Comment:
		fmt.Printf("record comment: %v\n", v)
	case Gift:
		fmt.Printf("record gift: %v\n", v)
	case nil:
	default:
	}
}

type wsConnection struct {
	URL url.URL
}

func connect(ctx context.Context, w *wsConnection) {
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
	log.Printf("[websocket] dial %s (%v)", w.URL.String(), header)
	c, _, err := dialer.DialContext(ctx, w.URL.String(), header)
	if err != nil {
		panic(err)
	}
	log.Printf("[websocket] connected")

	// commands
	subcmd := []byte(fmt.Sprintf("SUB\t%s", bcsvrKey))
	pingcmd := []byte("PING\tshowroom")
	// quitcmd := []byte("QUIT")

	// subscribe
	log.Printf("[websocket] %s", subcmd)
	if err := c.WriteMessage(websocket.TextMessage, subcmd); err != nil {
		log.Println("[websocket] tried to send sub:", err)
	}

	done := make(chan struct{}, 1)
	// read
	track.Add(1)
	go func() {
		defer track.Done()
		defer close(done)
		log.Printf("[websocket] read")
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("[websocket:read] err:", err)
				return
			}
			RecordEvent(msg)
		}
	}()

	// write
	track.Add(1)
	go func() {
		defer track.Done()

		log.Printf("[websocket] write")
		defer c.Close()

		pingTicker := time.NewTicker(1 * time.Minute)
		defer pingTicker.Stop()
		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
				log.Printf("[websocket:write] %s", pingcmd)
				if err := c.WriteMessage(websocket.TextMessage, pingcmd); err != nil {
					log.Println("[websocket:write] tried to send ping:", err)
				}
			case <-ctx.Done():
				/*
					// sending quit causes abnormal closure
					log.Printf("[websocket] %s", quitcmd)
					if err := c.WriteMessage(websocket.TextMessage, quitcmd); err != nil {
						log.Println("[websocket:write] tried to send quit:", err)
					}
				*/

				log.Println("[websocket:write] close...")
				if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
					log.Println("[websocket:write] close:", err)
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

// WatchEvents tracks websockets events
func WatchEvents(ctx context.Context) {
	// create websocket connection
	connect(ctx, &wsConnection{
		URL: url.URL{
			Scheme: "wss",
			Host:   bcsvrHost,
		},
	})

	track.Add(1)
	go func() {
		defer track.Done()

		for {
			select {
			case <-ctx.Done():
				log.Println("showroom.WatchEvents:", ctx.Err())
				return
			default:
			}
		}
	}()

	return
}
