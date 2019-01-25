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

var timeout = 5 * time.Minute
var errSnipeTimeout = errors.New("track.snipe: timeout")
var errSnipeNotFound = errors.New("track.snipe: did not find a stream url")

var sniping = struct {
	sync.RWMutex
	lookup map[string][]time.Time
}{
	lookup: make(map[string][]time.Time),
}

func hasSnipe(link string, at time.Time) bool {
	sniping.RLock()
	defer sniping.RUnlock()
	if lst, ok := sniping.lookup[link]; ok {
		for _, t := range lst {
			if t == at {
				return true
			}
		}
	}

	return false
}

// give true if it is newly added
func addSnipe(link string, at time.Time) bool {
	sniping.Lock()
	defer sniping.Unlock()

	if _, ok := sniping.lookup[link]; ok {
		return false
	}

	sniping.lookup[link] = append(sniping.lookup[link], at)
	return true
}

func delSnipe(link string, at time.Time) {
	sniping.Lock()
	defer sniping.Unlock()
	lst, ok := sniping.lookup[link]
	if !ok {
		return
	}
	for ndx, t := range lst {
		if t == at {
			sniping.lookup[link] = append(lst[:ndx], lst[ndx:]...)
		}
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

	upcomingAt := tracked.UpcomingAt()
	if at == upcomingAt {
		log.Println("track.SnipeAt:", tracked.Name(), "already sniping at given time")
		return nil
	} else if !upcomingAt.IsZero() {
		log.Println("track.SnipeAt:", tracked.Name(), "sniping new time")
	}

	go snipe(ctx, tracked, at)

	return nil
}

// snipe a stream that may have ended
func snipeMaybeEnded(ctx context.Context, tracked *tracked) error {
	name := tracked.Name()
	link := tracked.Link()

	log.Println("track.snipeMaybeEnded:", name)
	err := snipe(ctx, tracked, time.Now())

	if err == nil {
		// we found a stream so we allow a new save
		log.Println("track.snipeMaybeEnded:", name, "recovered")
		delSave(link)
		if err := Save(ctx, tracked); err != nil {
			log.Println("track.snipeMaybeEnded:", err)
			return err
		}
	}

	// we were trying to save but we timed out or did not find a url
	if err == errSnipeNotFound || err == errSnipeTimeout {
		if err == errSnipeTimeout {
			// so we were finished minutes ago
			at := time.Now().Add(-timeout)
			tracked.SetFinishedAt(at)
			log.Printf("track.snipeMaybeEnded: %s was finished at %s", name, at)
		} else if err == errSnipeNotFound {
			// no stream url so consider us finished now
			tracked.SetFinishedAt(time.Now())
			log.Printf("track.snipeMaybeEnded: %s finished now", name)
		}
	}

	return err
}

func snipe(ctx context.Context, tracked *tracked, upcomingAt time.Time) (err error) {
	wg.Add(1)
	defer wg.Done()

	name := tracked.Name()
	link := tracked.Link()

	addSnipe(link, upcomingAt)
	tracked.BeginSnipe()
	log.Println("track.snipe:", name)

	// wait until we expect the target to stream
	check := time.NewTimer(time.Until(upcomingAt))
	defer check.Stop()

	// clean up snipe
	defer func() {
		delSnipe(link, upcomingAt)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("track.snipe:", name, ctx.Err())
			return
		case <-check.C:
			to := time.NewTimer(timeout)
			defer to.Stop()

			var ok bool

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

			if hasSave(link) {
				// already saving so never mind
				log.Println("track.snipe:", name, "already saving")
				return
			}

			if err = Save(ctx, tracked); err != nil {
				log.Println("track.snipe:", err)
				return
			}

			return
		}
	}
}
