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

package homedir

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var cache string
var rw sync.RWMutex

// Dir gives the user's home directory
func Dir() (dir string, err error) {
	rw.RLock()
	c := cache
	rw.RUnlock()

	if c != "" {
		return c, nil
	}

	defer func() {
		if err == nil {
			rw.Lock()
			cache = dir
			rw.Unlock()
		}
	}()

	// try environment
	if dir = os.Getenv("HOME"); dir != "" {
		return
	}

	var stdout bytes.Buffer

	// try dscl
	cmd := exec.Command("sh", "-c", `dscl -q . -read /Users/"$(whoami)" NFSHomeDirectory | sed 's/^[^ ]*: //'`)
	cmd.Stdout = &stdout
	if err = cmd.Run(); err == nil {
		if dir = strings.TrimSpace(stdout.String()); dir != "" {
			return
		}
	}

	// fallback to shell
	stdout.Reset()
	cmd = exec.Command("sh", "-c", "cd && pwd")
	cmd.Stdout = &stdout
	if err = cmd.Run(); err == nil {
		if dir = strings.TrimSpace(stdout.String()); dir != "" {
			return
		}
	}

	err = errors.New("homedir.Dir: could not find home directory")
	return
}

// Expand tilde in a path
func Expand(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", fmt.Errorf("homedir.Expand: cannot expand path: '%s'", path)
	}

	dir, err := Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, path[1:]), nil
}
