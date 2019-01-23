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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
	"github.com/fsnotify/fsnotify"
)

// User in showroom
type User struct {
	Name string
	ID   int
}

// Comment from chat
type Comment struct {
	User
	Text string
	At   time.Time
}

// Gift sent
type Gift struct {
	User
	ID     int
	Amount int
	At     time.Time
}

var m sync.RWMutex

var targets = make([]Target, 0)

func check(ctx context.Context) error {
	if len(targets) == 0 {
		log.Println("showroom.check: no targets")
		return nil
	}
	log.Println("showroom.check:", len(targets), "targets")

	m.RLock()
	defer m.RUnlock()
	var waitCheck sync.WaitGroup
	for _, target := range targets {
		waitCheck.Add(1)
		go func(t Target) {
			defer waitCheck.Done()

			// each target gets a separate timeout
			// check is called by poll so we only check for a little while
			timeout := time.NewTimer(7 * time.Second)
			defer timeout.Stop()

			// check target's actual room for stream url or upcoming date
			var streamURL string
			var err error
			if streamURL, err = t.CheckStream(ctx); err == nil {
				log.Println("showroom.check:", t.name, "is live now!", streamURL)
				// they are live now so snipe them now
				if err = track.SnipeTargetAt(ctx, t, time.Now()); err != nil {
					log.Println("showroom.check:", err)
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
					streamURL, err = t.checkRoom()
					if err == nil {
						if err = track.SnipeTargetAt(ctx, t, time.Now()); err != nil {
							log.Println("showroom.check:", err)
						}
					}
				case <-timeout.C:
					log.Println("showroom.check:", t.name, "timeout")
					return
				case <-ctx.Done():
					log.Println("showroom.check:", t.name, ctx.Err())
					return
				}
			}
		}(target)
	}

	// wait for each target to finish checking
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		waitCheck.Wait()
	}()
	select {
	case <-done:
		log.Println("showroom.check: done")
	case <-ctx.Done():
		log.Println("showroom.check:", ctx.Err())
	}

	return nil
}

// Start showroom module
func Start(ctx context.Context) (err error) {
	// clean shutdown
	go func() {
		<-ctx.Done()
		log.Println("showroom: finishing...")
		track.Wait()
	}()

	// read the track list to find out who we are watching
	if err = readTrackList(ctx); err != nil {
		err = fmt.Errorf("showroom.Start: %s", err)
		return
	}

	select {
	case <-ctx.Done():
		log.Println("showroom.Start:", ctx.Err())
		return
	default:
	}

	if err = track.Poll(ctx, check); err != nil {
		err = fmt.Errorf("showroom.Start: %s", err)
		return
	}

	select {
	case <-ctx.Done():
		log.Println("showroom.Start:", ctx.Err())
		return
	default:
	}

	// watch track list
	track.Add(1)
	go func() {
		defer track.Done()
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.Println("showroom.Start: cannot make watcher:", err)
			return
		}

		if err := w.Add(track.ListPath); err != nil {
			log.Println("showroom.Start: cannot watch track list:", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				log.Println("showroom.Start:", ctx.Err())
				return
			case ev := <-w.Events:
				log.Println("showroom.Start: update:", ev.Name, ev.Op)
				if ev.Op == fsnotify.Write || ev.Op == fsnotify.Remove {
					readTrackList(ctx)
				}
			case err := <-w.Errors:
				log.Println("showroom.Start: error:", err)
			}
		}
	}()

	log.Println("showroom.Start: ok")

	return
}

func readTrackList(ctx context.Context) error {
	log.Println("showroom.readTrackList: reading...")

	f, err := os.Open(track.ListPath)
	if err != nil {
		return fmt.Errorf("showroom.readTrackList: %s", err)
	}
	defer f.Close()

	// read valid urls from track list
	s := bufio.NewScanner(f)
	lst := make(map[string]bool, len(targets))
	for s.Scan() {
		url := strings.TrimSpace(s.Text())
		if url == "" || url[0] == '#' {
			continue
		}
		lst[url] = true
	}

	// remove missing targets
	for _, t := range targets {
		if _, ok := lst[t.link]; !ok {
			ok, err := RemoveTargetFromURL(t.link)
			if err != nil {
				fmt.Println("showroom:", err)
				continue
			}
			if ok {
				fmt.Println("showroom: removed", t.link)
			}
		}
	}

	// add targets
	var waitAdd sync.WaitGroup
	for url := range lst {
		select {
		case <-ctx.Done():
			log.Println("showroom.readTrackList:", ctx.Err())
			break
		default:
		}
		waitAdd.Add(1)
		go func(u string) {
			defer waitAdd.Done()
			ok, err := AddTargetFromURL(ctx, u)
			if err != nil {
				fmt.Println("showroom:", err)
				return
			}
			if ok {
				fmt.Println("showroom: added", u)
			}

			return
		}(url)
	}

	// wait until all urls have been added
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		waitAdd.Wait()
	}()
	select {
	case <-done:
		log.Println("showroom.readTrackList: done")
	case <-ctx.Done():
		log.Println("showroom.readTrackList:", ctx.Err())
	}

	return nil
}
