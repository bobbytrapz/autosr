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

package dashboard

import (
	"fmt"
	"os/exec"
)

func openLink(link string) error {
	var cmd *exec.Cmd
	app := "open"
	cmd = exec.Command(app, link)

	app, err := exec.LookPath(app)
	if err != nil {
		return fmt.Errorf("openLink: could not find: %s: %s", app, err)
	}

	return cmd.Run()
}
