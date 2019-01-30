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
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var rw sync.RWMutex
var tracking = make(map[string]*tracked)

func getTracked(link string) (tracked *tracked, err error) {
	rw.RLock()
	defer rw.RUnlock()

	tracked = tracking[link]

	if tracked == nil {
		err = fmt.Errorf("track.getTracked: did not find: %s", link)
		return
	}

	return
}

var wg sync.WaitGroup

// Add a task
func Add(delta int) {
	wg.Add(delta)
}

// Done removes a task
func Done() {
	wg.Done()
}

// Start tracking
func Start(ctx context.Context) error {
	// read the track list to find out who we are watching
	if err := readList(ctx); err != nil {
		err = fmt.Errorf("track.Start: %s", err)
		return err
	}

	if err := beginPoll(ctx); err != nil {
		err = fmt.Errorf("track.Start: %s", err)
		return err
	}

	// watch track list
	Add(1)
	go func() {
		defer Done()
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.Println("track.Start: cannot make watcher:", err)
			return
		}

		if err := w.Add(listPath); err != nil {
			log.Println("track.Start: cannot watch track list:", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				log.Println("track.Start:", ctx.Err())
				return
			case ev := <-w.Events:
				log.Println("track.Start: update:", ev.Name, ev.Op)
				if ev.Op == fsnotify.Write || ev.Op == fsnotify.Remove {
					readList(ctx)
				}
			case err := <-w.Errors:
				log.Println("track.Start: error:", err)
			}
		}
	}()

	return nil
}

// Wait for tracking tasks to finish
func Wait() {
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		wg.Wait()
	}()
	log.Println("track.Wait: finishing...")
	select {
	case <-time.After(5 * time.Second):
		log.Println("track.Wait: force shutdown")
		os.Exit(0)
	case <-done:
		log.Println("track.Wait: done")
		return
	}
}

// AddTarget for tracking
func AddTarget(ctx context.Context, link string) error {
	rw.RLock()
	_, ok := tracking[link]
	rw.RUnlock()
	if ok {
		return nil
	}

	u, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("track.AddTarget: %s %s", link, err)
	}

	host := u.Hostname()
	m, err := FindModule(host)
	if err != nil {
		return err
	}

	target, err := m.AddTarget(ctx, link)
	if err != nil {
		return fmt.Errorf("track.AddTarget: %s %s", link, err)
	}
	if target == nil {
		return errors.New("track.AddTarget: target is nil")
	}

	rw.Lock()
	fmt.Println(host, "added", link)
	added := &tracked{
		target: target,
	}
	tracking[link] = added
	rw.Unlock()

	// check target right away
	if _, err := target.CheckStream(ctx); err == nil {
		log.Println("track.AddTarget:", target.Name(), "is live now!")
		// they are live now so try to snipe them now
		if err = SnipeAt(ctx, added, time.Now()); err != nil {
			log.Println("track.AddTarget:", err)
		}
	}

	return nil
}

// RemoveTarget from tracking
func RemoveTarget(ctx context.Context, link string) error {
	rw.RLock()
	tracked, ok := tracking[link]
	rw.RUnlock()

	if !ok {
		return errors.New("track.RemoveTarget: we are not tracking this target")
	}

	u, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("track.RemoveTarget: %s %s", link, err)
	}

	host := u.Hostname()
	m, err := FindModule(host)
	if err != nil {
		return err
	}

	target, err := m.RemoveTarget(ctx, link)
	if err != nil {
		return fmt.Errorf("track.RemoveTarget: %s %s", link, err)
	}
	if target == nil {
		return errors.New("track.RemoveTarget: target is nil")
	}

	rw.Lock()
	delete(tracking, link)
	rw.Unlock()

	tracked.Cancel()
	fmt.Println(host, "removed", link)

	return nil
}

// CancelTarget processing
func CancelTarget(link string) error {
	rw.Lock()
	tracked, ok := tracking[link]
	rw.Unlock()

	if !ok {
		return fmt.Errorf("track.CancelTarget: did not find: %s", link)
	}

	tracked.Cancel()

	return nil
}
