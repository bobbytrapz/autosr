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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/options"
	"github.com/fsnotify/fsnotify"
)

var rw sync.RWMutex
var tracking = make(map[string]*tracked)

func beginTracking(t *tracked) {
	rw.Lock()
	defer rw.Unlock()
	tracking[t.Link()] = t
}

func getTracking(link string) *tracked {
	rw.RLock()
	defer rw.RUnlock()
	if t, ok := tracking[link]; ok {
		return t
	}
	return nil
}

func endTracking(link string) (removed *tracked) {
	rw.Lock()
	defer rw.Unlock()
	removed = tracking[link]
	delete(tracking, link)
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
		defer w.Close()

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
	if getTracking(link) != nil {
		// silently ignore attempt to add a target we already have
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

	// if there was an error but the link was valid the module should return a target
	// if the module is unable to handle this link it should return nil
	if target == nil {
		return fmt.Errorf("track.AddTarget: link was not accepted: %q", link)
	}

	fmt.Println(host, "added", link)
	added := &tracked{
		target:   target,
		cancel:   make(chan struct{}),
		hostname: host,
	}
	beginTracking(added)

	if err != nil {
		return fmt.Errorf("track.AddTarget: %s %s", link, err)
	}

	// check target right away
	if _, err := target.CheckStream(ctx); err == nil {
		log.Println("track.AddTarget:", target.Name(), "is live now!")
		// they are live now so try to snipe them now
		if err = snipeAt(ctx, added, time.Now()); err != nil {
			log.Println("track.AddTarget:", err)
		}
	}

	return nil
}

// RemoveTarget from tracking
func RemoveTarget(ctx context.Context, link string) error {
	if getTracking(link) == nil {
		return errors.New("track.RemoveTarget: we are not tracking this target")
	}

	u, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("track.RemoveTarget: %s %s", link, err)
	}

	host := u.Hostname()
	if t := endTracking(link); t != nil {
		t.Cancel()
		fmt.Println(host, "removed", link)
	}

	return nil
}

// CancelTarget processing
func CancelTarget(link string) error {
	t := getTracking(link)
	if t == nil {
		return fmt.Errorf("track.CancelTarget: did not find: %s", link)
	}
	t.Cancel()

	return nil
}

// data is a json string
func runHooks(name string, data map[string]interface{}) {
	log.Println("track.runHooks:", name, data)

	hooksDir := filepath.Join(options.ConfigPath, "hooks", name)
	files, err := ioutil.ReadDir(hooksDir)
	if err != nil {
		log.Println("track.runHooks:", err)
		return
	}

	m, err := json.Marshal(data)
	if err != nil {
		log.Println("track.runHooks:", err)
		return
	}
	arg := string(m)

	for _, f := range files {
		if f.Mode()&0111 != 0 {
			log.Println("track.runHooks: execute", f.Name())
			cmdpath := filepath.Join(hooksDir, f.Name())
			cmd := exec.Command(cmdpath, arg)
			cmd.Dir = hooksDir
			err := cmd.Start()
			if err != nil {
				log.Println("track.runHooks:", err)
				continue
			}
		}
	}

	return
}
