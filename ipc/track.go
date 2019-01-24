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
