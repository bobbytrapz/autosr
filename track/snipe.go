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

var errSnipeTimeout = errors.New("track.snipe: timeout")
var errSnipeNotFound = errors.New("track.snipe: did not find a stream url")

var sniping = struct {
	sync.RWMutex
	lookup map[string]chan struct{}
}{
	lookup: make(map[string]chan struct{}),
}

func hasSnipe(link string) bool {
	sniping.RLock()
	defer sniping.RUnlock()
	_, ok := sniping.lookup[link]
	return ok
}

// give true if it is newly added
func addSnipe(link string, cancel chan struct{}) bool {
	sniping.Lock()
	defer sniping.Unlock()

	if _, ok := sniping.lookup[link]; ok {
		return false
	}

	sniping.lookup[link] = cancel
	return true
}

func delSnipe(link string) {
	sniping.Lock()
	defer sniping.Unlock()
	if cancel, ok := sniping.lookup[link]; ok {
		close(cancel)
	}
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
		log.Println("track.SnipeAt:", tracked.Name(), "already saving")
		return nil
	}

	if hasSnipe(link) {
		if at == tracked.UpcomingAt() {
			log.Println("track.SnipeAt:", tracked.Name(), "already sniping at given time")
			return nil
		}

		log.Println("track.SnipeAt:", tracked.Name(), "sniping updated time")
		delSnipe(link)
	}
	tracked.SetUpcomingAt(at)

	return snipe(ctx, tracked)
}

// snipe a stream we think has ended
// we do not check if it is saving
func snipeEnded(ctx context.Context, tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.snipeEnded: invalid time")
	}

	name := tracked.Name()
	link := tracked.Link()

	if hasSnipe(link) {
		log.Println("track.snipeEnded:", name, "already sniping")
		// so consider this stream has finished
		tracked.SetFinishedAt(at)
		log.Printf("track.snipe: %s finished at %s", name, at)
		// end save
		delSave(link)
		tracked.EndSave(nil)
		return nil
	}
	tracked.SetUpcomingAt(at)

	return snipe(ctx, tracked)
}

func snipe(ctx context.Context, tracked *tracked) error {
	tracked.BeginSnipe()
	upcomingAt := tracked.UpcomingAt()
	cancelSnipe := make(chan struct{}, 1)

	name := tracked.Name()
	link := tracked.Link()

	addSnipe(link, cancelSnipe)

	// snipe target
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("track.snipe:", name)

		// wait until we expect the target to stream
		check := time.NewTimer(time.Until(upcomingAt))
		defer check.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("track.snipe:", name, ctx.Err())
				return
			case <-cancelSnipe:
				log.Println("track.snipe:", name, "cancelled")
				return
			case <-check.C:
				// set timeout for sniping
				timeout := 5 * time.Minute
				to := time.NewTimer(timeout)
				defer to.Stop()

				var ok bool
				var err error

				// handle finishing saves
				defer func() {
					// we are saving but we timed out or did not find a url
					if hasSave(link) && (err == errSnipeNotFound || err == errSnipeTimeout) {
						if err == errSnipeTimeout {
							// so we were finished minutes ago
							at := time.Now().Add(-timeout)
							tracked.SetFinishedAt(at)
							log.Printf("track.snipe: %s was finished at %s", name, at)
						} else if err == errSnipeNotFound {
							// no stream url so consider us finished now
							tracked.SetFinishedAt(time.Now())
							log.Printf("track.snipe: %s finished now", name)
						}

						// end save
						delSave(link)
						tracked.EndSave(nil)
					}
				}()

				// check if the user is online
				var isLive bool
				var liveErr retry.BoolRetryable
				numAttempts := 0
				isLive, err = tracked.CheckLive(ctx)
				liveErr, ok = retry.BoolCheck(err)
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
						err = errSnipeTimeout
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
				urlErr, ok = retry.StringCheck(err)
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
						log.Println("track.snipe:", name, "timeout while looking for stream url")
						err = errSnipeTimeout
						return
					}
				}

				if err != nil {
					// we failed to find something to save
					err = errSnipeNotFound
					log.Println("track.snipe:", name, "did not find url")
					return
				}

				log.Println("track.snipe:", name, "found url.")
				tracked.SetStreamURL(url)
				if err := Save(ctx, tracked); err != nil {
					log.Println("track.snipe:", err)
					return
				}

				// saving so end all sniping for this link
				delSnipe(link)

				return
			}
		}
	}()

	return nil
}
