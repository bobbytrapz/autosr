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
)

// Module is called by poll and add/remove target
type Module interface {
	// information
	Hostname() string
	// called by track.poll
	CheckUpcoming(context.Context) error
	// called by track.Add/RemoveTarget
	// give the target we added or removed or nil
	AddTarget(ctx context.Context, link string) (Target, error)
	RemoveTarget(ctx context.Context, link string) (Target, error)
}

var modules = make(map[string]Module)

// RegisterModule with a hostname
// note: we do not expect to be called after tracking begins
func RegisterModule(m Module) error {
	hostname := m.Hostname()

	if _, ok := modules[hostname]; ok {
		return errors.New("track.RegisterModule: hostname already registered")
	}

	modules[hostname] = m

	return nil
}

// FindModule with hostname
func FindModule(hostname string) (m Module, err error) {
	var ok bool
	if m, ok = modules[hostname]; ok {
		return
	}

	err = errors.New("track.FindModule: module not found")
	return
}
