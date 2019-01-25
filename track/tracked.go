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
	"sync"
	"time"
)

type tracked struct {
	sync.RWMutex
	target     Target
	cancelSave chan struct{}

	// schelduling
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

func (t *tracked) EndSave() {
	t.RLock()
	defer t.RUnlock()
	if t.target == nil {
		return
	}

	t.target.EndSave()
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
	return time.Until(t.UpcomingAt().Add(snipeTimeout)) > 0
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

// StartedAt for target
func (t *tracked) StartedAt() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.startedAt
}

// SetStartedAt for target
func (t *tracked) SetStartedAt(at time.Time) error {
	t.Lock()
	defer t.Unlock()

	if t.target == nil {
		return errors.New("track.SetStartedAt: target is nil")
	}

	if err := addSave(t.target.Link()); err != nil {
		return err
	}

	t.startedAt = at

	t.target.BeginSave()

	return nil
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

	if t.target == nil {
		return
	}

	delSave(t.target.Link())

	t.finishedAt = at

	t.target.EndSave()

	return
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
