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
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
	"github.com/bobbytrapz/autosr/track"
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

var cancel context.CancelFunc

// Start showroom module
func Start() (err error) {
	cancel, err = track.Poll(update)

	return
}

// Stop showroom module
func Stop() {
	cancel()
}
