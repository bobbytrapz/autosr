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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

var m sync.RWMutex
var tracking = make(map[string]*tracked)

var wg sync.WaitGroup

// Add a task
func Add(delta int) {
	wg.Add(delta)
}

// Done removes a task
func Done() {
	wg.Done()
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

type tracked struct {
	sync.RWMutex
	target     Target
	cancelSave chan struct{}

	// schelduling
	upcomingAt time.Time
	startedAt  time.Time
	finishedAt time.Time

	// recording
	streamURL string
}

func (t *tracked) Display() string {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return ""
	}

	return t.target.Display()
}

func (t *tracked) Name() string {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return ""
	}

	return t.target.Name()
}

func (t *tracked) Link() string {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return ""
	}

	return t.target.Link()
}

func (t *tracked) BeginSnipe() {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return
	}

	t.target.BeginSnipe()
}

func (t *tracked) BeginSave() {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return
	}

	t.target.BeginSave()
}

func (t *tracked) EndSave(err error) {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return
	}

	t.target.EndSave(err)
}

func (t *tracked) CheckLive(ctx context.Context) (bool, error) {
	if t.target != nil {
		return t.target.CheckLive(ctx)
	}

	return false, errors.New("target is nil")
}

func (t *tracked) CheckStream(ctx context.Context) (string, error) {
	if t.target != nil {
		return t.target.CheckStream(ctx)
	}

	return "", errors.New("target is nil")
}

func (t *tracked) Cancel() {
	if t.cancelSave != nil {
		close(t.cancelSave)
	}
}

// SetCancel for tracked streamer
func (t *tracked) SetCancel(ch chan struct{}) {
	t.Lock()
	defer t.Unlock()
	t.cancelSave = ch
}

// IsUpcoming is true if the target has a known upcoming time
func (t *tracked) IsUpcoming() bool {
	return time.Until(t.UpcomingAt().Add(5*time.Minute)) > 0
}

// IsLive is true if the target is live
func (t *tracked) IsLive() bool {
	return !t.StartedAt().IsZero() && t.StartedAt().After(t.FinishedAt())
}

// IsFinished is true if the target stream has ended
func (t *tracked) IsFinished() bool {
	return !t.FinishedAt().IsZero() && t.StartedAt().Before(t.FinishedAt())
}

// IsOffline is true when the stream is not live and we do not when it will be live
func (t *tracked) IsOffline() bool {
	return !t.IsLive() && !t.IsUpcoming()
}

// AddTarget for tracking
func AddTarget(target Target) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := tracking[target.Link()]; ok {
		return errors.New("track.AddTarget: we are already tracking this target")
	}

	tracking[target.Link()] = &tracked{
		target: target,
	}

	return nil
}

// RemoveTarget from tracking
func RemoveTarget(link string) error {
	m.Lock()
	defer m.Unlock()

	tracked, ok := tracking[link]
	if !ok {
		return errors.New("track.RemoveTarget: we are not tracking this target")
	}

	tracked.Cancel()

	delete(tracking, link)

	return nil
}

// CancelTarget processing
func CancelTarget(link string) error {
	m.Lock()
	defer m.Unlock()

	tracked, ok := tracking[link]
	if !ok {
		return errors.New("track.CancelTarget: we are not tracking this target")
	}

	tracked.Cancel()

	return nil
}

// UpcomingAt for target
func (t *tracked) UpcomingAt() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.upcomingAt
}

// SetUpcomingAt for target
func (t *tracked) SetUpcomingAt(at time.Time) {
	t.Lock()
	defer t.Unlock()
	t.upcomingAt = at
}

// StartedAt for target
func (t *tracked) StartedAt() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.startedAt
}

// SetStartedAt for target
func (t *tracked) SetStartedAt(at time.Time) {
	t.Lock()
	defer t.Unlock()
	t.startedAt = at
}

// FinishedAt for target
func (t *tracked) FinishedAt() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.finishedAt
}

// SetFinishedAt for target
func (t *tracked) SetFinishedAt(at time.Time) {
	t.Lock()
	defer t.Unlock()
	t.finishedAt = at
}

func (t *tracked) StreamURL() string {
	t.RLock()
	defer t.RUnlock()
	return t.streamURL
}

// SetStreamURL for target
func (t *tracked) SetStreamURL(url string) {
	t.Lock()
	defer t.Unlock()
	t.streamURL = url
}

func getTracked(link string) (tracked *tracked, err error) {
	m.RLock()
	defer m.RUnlock()

	tracked = tracking[link]

	if tracked == nil {
		err = errors.New("track.GetTarget: we are not tracking this target")
		return
	}

	return
}

// ListPath to list of urls to watch
var ListPath = filepath.Join(options.ConfigPath, "track.list")
