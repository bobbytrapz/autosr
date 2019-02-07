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
	"log"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/backoff"
	"github.com/bobbytrapz/autosr/retry"
)

var snipeTimeout = 15 * time.Minute
var errSnipeTimeout = errors.New("track.snipe: timeout")
var errSnipeNotFound = errors.New("track.snipe: did not find a stream url")

type snipeTask struct {
	name string
	link string
	at   time.Time
}

var sniping = struct {
	sync.RWMutex
	tasks map[snipeTask]time.Time
}{
	tasks: make(map[snipeTask]time.Time),
}

// find the most recent snipe for a link
func findSnipeTask(link string) (task snipeTask, createdAt time.Time) {
	sniping.RLock()
	defer sniping.RUnlock()

	// find the most recently added snipe task matching our target
	for t, at := range sniping.tasks {
		if t.link == link {
			if createdAt.IsZero() || at.After(createdAt) {
				createdAt = at
				task = t
			}
		}
	}

	return
}

func hasSnipeTask(task snipeTask) bool {
	sniping.RLock()
	defer sniping.RUnlock()
	_, ok := sniping.tasks[task]
	return ok
}

// give true if it is newly added
func addSnipeTask(task snipeTask) bool {
	if hasSnipeTask(task) {
		return false
	}
	sniping.Lock()
	defer sniping.Unlock()
	sniping.tasks[task] = time.Now()
	return true
}

func delSnipeTask(task snipeTask) {
	sniping.Lock()
	defer sniping.Unlock()
	delete(sniping.tasks, task)
}

// SnipeTargetAt snipes a target at the given time
func SnipeTargetAt(ctx context.Context, t Target, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeTargetAt: invalid time")
	}

	tracked := getTracking(t.Link())
	if tracked == nil {
		return errors.New("track.SnipeTarget: invalid target")
	}

	return snipeAt(ctx, tracked, at)
}

// snipes a target at the given time
func snipeAt(ctx context.Context, tracked *tracked, at time.Time) error {
	if at.IsZero() {
		return errors.New("track.SnipeAt: invalid time")
	}
	log.Println("track.SnipeAt:", tracked.Name(), "at", at.Format(time.UnixDate))
	go performSnipe(ctx, tracked, at)
	return nil
}

func performSnipe(ctx context.Context, t *tracked, upcomingAt time.Time) (err error) {
	wg.Add(1)
	defer wg.Done()

	task := snipeTask{
		name: t.Name(),
		link: t.Link(),
		at:   upcomingAt,
	}
	if !addSnipeTask(task) {
		log.Println("track.snipe: already sniping", task.name, "at", task.at)
		return
	}
	defer func() {
		delSnipeTask(task)
	}()
	t.BeginSnipe()
	log.Println("track.snipe:", task.name)

	// wait until we expect the target to stream
	check := time.NewTimer(time.Until(upcomingAt))
	defer check.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("track.snipe:", task.name, ctx.Err())
			return
		case <-check.C:
			err = waitForLive(ctx, t, snipeTimeout)
			if err != nil {
				return
			}

			log.Println("track.snipe:", task.name, "is online")

			var streamURL string
			streamURL, err = waitForStream(ctx, t, snipeTimeout)
			if err != nil {
				// we failed to find a stream url
				log.Println("track.snipe:", task.name, "did not find url")
				return
			}

			log.Println("track.snipe:", task.name, "found url.")
			go performSave(ctx, t, streamURL)

			return
		}
	}
}

func waitForLive(ctx context.Context, t *tracked, timeout time.Duration) (err error) {
	to := time.NewTimer(timeout)
	defer to.Stop()

	name := t.Name()

	// check if the user is online
	var ok bool
	var isLive bool
	var liveErr retry.BoolRetryable
	numAttempts := 0
	isLive, err = t.CheckLive(ctx)
	if isLive {
		return
	}
	liveErr, ok = retry.BoolCheck(err)
	for ; ok; liveErr, ok = retry.BoolCheck(err) {
		select {
		case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
			numAttempts++
			isLive, err = liveErr.Retry()
			if isLive {
				return
			}
		case <-to.C:
			log.Println("track.waitForLive:", name, "timeout")
			err = errSnipeTimeout
			return
		case <-ctx.Done():
			log.Println("track.waitForLive:", name, ctx.Err())
			return
		}
	}

	return
}

func waitForStream(ctx context.Context, t *tracked, timeout time.Duration) (streamURL string, err error) {
	to := time.NewTimer(timeout)
	defer to.Stop()

	name := t.Name()

	// check to see if the target's stream has actually begun
	var ok bool
	var urlErr retry.StringRetryable
	numAttempts := 0
	streamURL, err = t.CheckStream(ctx)
	urlErr, ok = retry.StringCheck(err)
	for ; ok; urlErr, ok = retry.StringCheck(err) {
		select {
		case <-time.After(backoff.DefaultPolicy.Duration(numAttempts)):
			numAttempts++
			streamURL, err = urlErr.Retry()
			if err == nil {
				break
			}
			log.Println("track.waitForStream:", err)
		case <-ctx.Done():
			log.Println("track.waitForStream:", name, ctx.Err())
			return
		case <-to.C:
			log.Println("track.waitForStream:", name, "timeout while looking for stream url")
			err = errSnipeTimeout
			return
		}
	}

	return
}
