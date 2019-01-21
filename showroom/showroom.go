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
var wg sync.WaitGroup

// Wait for showroom tasks to finish
func Wait() {
	wg.Wait()
}

var targets = make([]Target, 0)

func update() error {
	if len(targets) == 0 {
		fmt.Println("showroom.update: no targets")
		return nil
	}

	var wg sync.WaitGroup
	m.RLock()
	for _, target := range targets {
		go func(t Target) {
			wg.Add(1)
			defer wg.Done()

			// each target gets a separate timeout
			timeout := time.NewTimer(time.Minute)
			defer timeout.Stop()

			isLive, err := checkIsLive(t.id)
			if err == nil && isLive {
				if err = track.SnipeTargetAt(t, time.Now()); err != nil {
					log.Println("showroom.update:", err)
				}
				return
			}

			numAttempts := 0
			e, ok := retry.BoolCheck(err)
			for ; ok; e, ok = retry.BoolCheck(err) {
				select {
				case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
					numAttempts++
					isLive, err = e.Retry()
					if err == nil && isLive {
						if err = track.SnipeTargetAt(t, time.Now()); err != nil {
							log.Println("showroom.update:", err)
						}
					}
				case <-timeout.C:
					log.Println("showroom.update:", t.name, "timeout")
					return
				}
			}
		}(target)
	}
	m.RUnlock()

	// wait for each target to finish checking
	wg.Wait()

	return nil
}

// Start showroom module
func Start(ctx context.Context) (err error) {
	// read the track list to find out who we are watching
	if err = readTrackList(); err != nil {
		err = fmt.Errorf("showroom.Start: %s", err)
		return
	}

	if err = track.Poll(ctx, update); err != nil {
		err = fmt.Errorf("showroom.Start: %s", err)
		return
	}

	// watch track list
	go func() {
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
				return
			case ev := <-w.Events:
				log.Println("showroom.Start: update:", ev.Name, ev.Op)
				if ev.Op == fsnotify.Write || ev.Op == fsnotify.Remove {
					readTrackList()
				}
			case err := <-w.Errors:
				log.Println("showroom.Start: error:", err)
			}
		}
	}()

	return
}

func readTrackList() error {
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
	var wg sync.WaitGroup
	for url := range lst {
		go func(u string) {
			wg.Add(1)
			defer wg.Done()
			ok, err := AddTargetFromURL(u)
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
	wg.Wait()
	fmt.Println("showroom.readTrackList: done")

	return nil
}
