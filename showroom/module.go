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
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
)

// Module is the showroom module
type Module struct{}

var module = Module{}

var rw sync.RWMutex
var targets = make([]target, 0)

func init() {
	if err := track.RegisterModule(module); err != nil {
		panic(err)
	}
}

// Hostname gives the hostname for this module
func (m Module) Hostname() string {
	return "www.showroom-live.com"
}

// AddTarget to track
func (m Module) AddTarget(ctx context.Context, link string) (track.Target, error) {
	_, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("showroom.AddTarget: '%s' %s", link, err)
	}

	s, err := fetchRoom(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("showroom.AddTarget: '%s' %s", link, err)
	}

	name := strings.TrimSpace(s.Name)
	// note: this works around a display bug in gocui
	var buf bytes.Buffer
	for _, r := range name {
		buf.WriteRune(r)
		if r != ' ' && r != '(' && r != ')' {
			buf.WriteRune(' ')
		}
	}
	display := buf.String()

	added := target{
		name:    name,
		display: display,
		id:      s.ID,
		link:    link,
		urlKey:  s.LiveRoom.URLKey,
	}

	rw.Lock()
	targets = append(targets, added)
	rw.Unlock()

	return added, nil
}

// RemoveTarget from tracking
func (m Module) RemoveTarget(ctx context.Context, link string) (track.Target, error) {
	var removed target

	rw.Lock()
	n := 0
	for ; n < len(targets); n++ {
		if targets[n].link == link {
			removed = targets[n]
			break
		}
	}
	if n < len(targets) {
		targets = append(targets[:n], targets[n+1:]...)
	}
	rw.Unlock()

	if removed.link != link {
		return nil, fmt.Errorf("showroom.RemoveTarget: did not find target for %s", link)
	}

	return &removed, nil
}

// CheckUpcoming streams and snipe them
func (m Module) CheckUpcoming(ctx context.Context) error {
	if len(targets) == 0 {
		log.Println("showroom.CheckUpcoming: no targets")
		return nil
	}
	log.Println("showroom.CheckUpcoming:", len(targets), "targets")

	rw.RLock()
	defer rw.RUnlock()
	var waitCheck sync.WaitGroup
	for _, tt := range targets {
		waitCheck.Add(1)
		go func(t target) {
			defer waitCheck.Done()

			// each target gets a separate timeout
			// check is called by poll so we only check for a little while
			timeout := time.NewTimer(7 * time.Second)
			defer timeout.Stop()

			// check target's actual room for stream url or upcoming date
			var err error
			if _, err = t.CheckStream(ctx); err == nil {
				log.Println("showroom.CheckUpcoming:", t.name, "is live now!")
				// they are live now so snipe them now
				if err = track.SnipeTargetAt(ctx, t, time.Now()); err != nil {
					log.Println("showroom.CheckUpcoming:", err)
				}
				return
			}

			numAttempts := 0
			e, ok := retry.StringCheck(err)
			for ; ok; e, ok = retry.StringCheck(err) {
				// check if error just a stream was not found or new time
				// if so, we should not bother retrying
				if !strings.HasPrefix(err.Error(), "showroom.CheckStream") {
					return
				}
				select {
				case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
					numAttempts++
					// check for room again
					_, err = e.Retry()
					if err == nil {
						if err = track.SnipeTargetAt(ctx, t, time.Now()); err != nil {
							log.Println("showroom.CheckUpcoming:", err)
						}
					}
				case <-timeout.C:
					log.Println("showroom.CheckUpcoming:", t.name, "timeout")
					return
				case <-ctx.Done():
					log.Println("showroom.CheckUpcoming:", t.name, ctx.Err())
					return
				}
			}
		}(tt)
	}

	// wait for each target to finish checking
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		waitCheck.Wait()
	}()
	select {
	case <-done:
		log.Println("showroom.CheckUpcoming: done")
	case <-ctx.Done():
		log.Println("showroom.CheckUpcoming:", ctx.Err())
	}

	return nil
}
