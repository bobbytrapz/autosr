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
)

// Target is being tracked for stream activity
type Target interface {
	// real name of streamer
	Name() string
	// for display in dashboard
	Display() string
	// url string
	Link() string

	// save path
	SavePath() string

	// check for live status
	CheckLive(context.Context) (bool, error)
	// check for a live stream
	CheckStream(context.Context) (string, error)

	// callback when sniping starts
	BeginSnipe(context.Context)
	// callback when save starts
	BeginSave(context.Context)
	// callback when save ends
	EndSave(context.Context)
}
