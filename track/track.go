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
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bobbytrapz/autosr/options"
)

type status int

const (
	_ = iota
	sleeping
	sniping
	saving
)

type tracked struct {
	sync.RWMutex
	Target Target
	cancel context.CancelFunc
	status status

	// schelduling
	upcomingAt time.Time
	startedAt  time.Time
	finishedAt time.Time

	// recording
	streamURL string
}

func (t *tracked) Cancel() {
	if t.cancel == nil {
		return
	}
	t.cancel()
}

// SetCancel for tracked streamer
func (t *tracked) SetCancel(c context.CancelFunc) {
	t.Lock()
	defer t.Unlock()
	t.cancel = c
}

// IsUpcoming is true if the target has a known upcoming time
func (t *tracked) IsUpcoming() bool {
	return !t.upcomingAt.IsZero()
}

// IsLive is true if the target is live
func (t *tracked) IsLive() bool {
	return !t.startedAt.IsZero() && t.finishedAt.IsZero()
}

// IsFinished is true if the target stream has ended
func (t *tracked) IsFinished() bool {
	return !t.finishedAt.IsZero()
}

// Status for tracked streamer
func (t *tracked) Status() status {
	t.RLock()
	defer t.RUnlock()
	return t.status
}

// SetStatus for tracked streamer
func (t *tracked) SetStatus(s status) {
	t.Lock()
	defer t.Unlock()
	t.status = s
}

var m sync.RWMutex
var tracking = make(map[string]*tracked)

var wg sync.WaitGroup

// Wait for tracking tasks to finish
func Wait() {
	wg.Wait()
}

// AddTarget for tracking
func AddTarget(target Target) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := tracking[target.Link()]; ok {
		return errors.New("track.AddTarget: we are already tracking this target")
	}

	tracking[target.Link()] = &tracked{
		Target: target,
		status: sleeping,
	}

	return nil
}

// RemoveTarget from tracking
func RemoveTarget(link string) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := tracking[link]; !ok {
		return errors.New("track.RemoveTarget: we are not tracking this target")
	}

	delete(tracking, link)

	return nil
}

// CancelTarget processing
func CancelTarget(link string) error {
	m.Lock()
	defer m.Unlock()

	t, ok := tracking[link]
	if !ok {
		return errors.New("track.CancelTarget: we are not tracking this target")
	}

	t.Cancel()

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

// ListTracking gives everyone we are tracking sorted by urgency
func ListTracking() []Info {
	m.RLock()
	defer m.RUnlock()

	var lst []Info
	for _, t := range tracking {
		lst = append(lst, Info{
			Name:       t.Target.Display(),
			Link:       t.Target.Link(),
			UpcomingAt: t.upcomingAt,
			StartedAt:  t.startedAt,
			FinishedAt: t.finishedAt,
		})
	}

	sort.Sort(byUrgency(lst))

	return lst
}

// ListPath to list of urls to watch
var ListPath = filepath.Join(options.ConfigPath, "track.list")
