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

package ipc

import (
	"fmt"
	"log"

	"github.com/bobbytrapz/autosr/track"
)

// CheckNow forces a poll attempt
func (c *Command) CheckNow(req *Dashboard, res *Dashboard) error {
	replicate(req, res)
	log.Println("ipc.CheckNow")
	track.CheckNow()
	return nil
}

// CancelTarget a target being sniped or saved
func (c *Command) CancelTarget(req *Dashboard, res *Dashboard) error {
	replicate(req, res)

	if status.SelectURL != "" {
		if err := track.CancelTarget(status.SelectURL); err != nil {
			return fmt.Errorf("ipc.CancelTarget: %s", err)
		}
	}

	return nil
}
