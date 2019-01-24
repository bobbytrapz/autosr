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
func SnipeTargetAt(ctx context.Context, t Target, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeTargetAt: invalid time")
	}

	tracked, err := getTracked(t.Link())
	if err != nil {
		return fmt.Errorf("track.SnipeTarget: %s", err)
	}

	return SnipeAt(ctx, tracked, at)
}

// SnipeAt snipes a target at the given time
func SnipeAt(ctx context.Context, tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}
	log.Println("track.SnipeAt:", tracked.Name(), "at", at.Format(time.UnixDate))

	link := tracked.Link()

	if hasSave(link) {
		log.Println("track.Snipe:", tracked.Name(), "already saving")
		return nil
	}

	addSnipe(link)
	tracked.SetUpcomingAt(at)

	return snipe(ctx, tracked)
}

// snipe a stream we think has ended
// we do not check if it is saving
func snipeEnded(ctx context.Context, tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}

	if !addSnipe(tracked.Link()) {
		log.Println("track.Snipe:", tracked.Name(), "already sniping")
		return nil
	}
	tracked.SetUpcomingAt(at)

	return snipe(ctx, tracked)
}

func snipe(ctx context.Context, tracked *tracked) error {
	tracked.BeginSnipe()
	upcomingAt := tracked.UpcomingAt()

	name := tracked.Name()
	link := tracked.Link()

	// snipe target
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("track.snipe:", name)
		defer delSnipe(link)

		// wait until we expect the target to stream
		check := time.NewTimer(time.Until(upcomingAt))
		defer check.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("track.snipe:", name, ctx.Err())
				return
			case <-check.C:
				// set timeout for sniping
				timeout := 5 * time.Minute
				to := time.NewTimer(timeout)
				defer to.Stop()

				var ok bool
				var err error

				// check if the user is online
				var isLive bool
				var liveErr retry.BoolRetryable
				numAttempts := 0
				isLive, err = tracked.CheckLive(ctx)
				for ; ok; liveErr, ok = retry.BoolCheck(err) {
					ok, err = liveErr.Retry()
					select {
					case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
						numAttempts++
						isLive, err = liveErr.Retry()
						if isLive {
							break
						}
					case <-to.C:
						log.Println("track.snipe:", name, "timeout")
						return
					case <-ctx.Done():
						log.Println("track.snipe:", name, ctx.Err())
						return
					}
				}
				log.Println("track.snipe:", name, "is online")

				// check to see if the target's stream has actually begun
				var url string
				var urlErr retry.StringRetryable
				numAttempts = 0
				url, err = tracked.CheckStream(ctx)
				for ; ok; urlErr, ok = retry.StringCheck(err) {
					select {
					case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
						numAttempts++
						url, err = urlErr.Retry()
						if err == nil {
							break
						}
						log.Println("track.snipe:", err)
					case <-ctx.Done():
						log.Println("track.snipe:", name, ctx.Err())
						return
					case <-to.C:
						link := tracked.Link()
						log.Println("track.snipe:", name, "timeout")
						if hasSave(link) {
							// so we were finished minutes ago
							at := time.Now().Add(-timeout)
							tracked.SetFinishedAt(at)
							log.Printf("track.snipe: %s finished at %s", name, at)
							// end save
							delSave(link)
							tracked.EndSave(nil)
						}
						return
					}
				}

				if err != nil {
					// we failed
					log.Println("track.snipe:", "did not find url:", err)
					tracked.SetUpcomingAt(time.Time{})
					return
				}

				log.Println("track.snipe:", name, "found url.")
				tracked.SetStreamURL(url)
				tracked.SetUpcomingAt(time.Time{})
				if err := Save(ctx, tracked); err != nil {
					log.Println("track.snipe:", err)
				}
				return
			}
		}
	}()

	return nil
}
