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
	"errors"
)

// Module is called by poll and add/remove target
type Module interface {
	// called by track.poll
	CheckUpcoming()
	// called by track.Add/RemoveTarget
	AddTarget()
	RemoveTarget()
}

var modules = make(map[string]Module)

// RegisterModule with a hostname
func RegisterModule(hostname string, m Module) error {
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
