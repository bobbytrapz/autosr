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

package track

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
)

var sniping = struct {
	sync.RWMutex
	lookup map[string]bool
}{
	lookup: make(map[string]bool),
}

func hasSnipe(link string) bool {
	sniping.RLock()
	defer sniping.RUnlock()
	return sniping.lookup[link]
}

// give true if it is newly added
func addSnipe(link string) bool {
	sniping.Lock()
	defer sniping.Unlock()

	if _, ok := sniping.lookup[link]; ok {
		return false
	}

	sniping.lookup[link] = true
	return true
}

func delSnipe(link string) {
	sniping.Lock()
	defer sniping.Unlock()
	delete(sniping.lookup, link)
}

// SnipeTargetAt snipes a target at the given time
func SnipeTargetAt(t Target, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeTargetAt: invalid time")
	}

	tracked, err := getTracked(t.Link())
	if err != nil {
		return fmt.Errorf("track.SnipeTarget: %s", err)
	}

	return SnipeAt(tracked, at)
}

// SnipeAt snipes a target at the given time
func SnipeAt(tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}
	log.Println("track.SnipeAt:", tracked.Target.Name(), "at", at.Format(time.UnixDate))

	link := tracked.Target.Link()

	if hasSave(link) {
		log.Println("track.Snipe:", tracked.Target.Name(), "already saving")
		return nil
	}

	if !addSnipe(link) {
		log.Println("track.Snipe:", tracked.Target.Name(), "already sniping")
		return nil
	}

	tracked.SetUpcomingAt(at)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	tracked.SetCancel(cancel)

	return snipe(ctx, tracked)
}

// snipe a stream we think has ended
// we do not check if it is saving
func snipeEnded(tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}

	if !addSnipe(tracked.Target.Link()) {
		log.Println("track.Snipe:", tracked.Target.Name(), "already sniping")
		return nil
	}

	tracked.SetUpcomingAt(at)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	tracked.SetCancel(cancel)

	return snipe(ctx, tracked)
}

func snipe(ctx context.Context, tracked *tracked) error {
	tracked.Target.BeginSnipe()
	upcomingAt := tracked.UpcomingAt()

	// snipe target
	go func() {
		wg.Add(1)
		defer wg.Done()

		log.Println("track.snipe:", tracked.Target.Name())
		defer delSnipe(tracked.Target.Link())

		// wait until we expect the target to stream
		check := time.NewTimer(time.Until(upcomingAt))
		defer check.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("track.snipe:", tracked.Target.Name(), "canceled")
				return
			case <-check.C:
				// set timeout for sniping
				timeout := 5 * time.Minute
				to := time.NewTimer(timeout)
				defer to.Stop()

				// check to see if the target's stream has actually begun
				url, err := tracked.Target.Check()
				if err != nil {
					if e, ok := retry.StringCheck(err); ok {
						// retry according to backoff policy
						for n := 0; ; n++ {
							select {
							case <-ctx.Done():
								log.Println("track.snipe:", tracked.Target.Name(), "canceled")
								return
							case <-to.C:
								link := tracked.Target.Link()
								log.Println("track.snipe:", tracked.Target.Name(), "timeout")
								if hasSave(link) {
									// so we were finished minutes ago
									at := time.Now().Add(-timeout)
									tracked.SetFinishedAt(at)
									log.Printf("track.snipe: %s finished at %s", tracked.Target.Name(), at)
									// end save
									delSave(link)
									tracked.Target.EndSave(nil)
								}
								return
							case <-time.After(backoff.DefaultPolicy.Duration(n)):
								url, err = e.Retry()
								if err == nil {
									break
								}
								e, ok = retry.StringCheck(err)
								if !ok {
									// we failed and should not try again
									return
								}
								log.Println("track.snipe:", err)
							}
						}
					}
				}
				// attempt ok
				log.Println("track.snipe:", tracked.Target.Name(), "found url.")
				tracked.SetStreamURL(url)
				tracked.SetUpcomingAt(time.Time{})
				// if we were saving forget it because we may have a new stream url
				delSave(tracked.Target.Link())
				if err := Save(ctx, tracked); err != nil {
					log.Println("track.snipe:", err)
				}
				return
			}
		}
	}()

	return nil
}
