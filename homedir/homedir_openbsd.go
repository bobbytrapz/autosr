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
	"strconv"
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

	// try getent
	cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
	cmd.Stdout = &stdout
	if err = cmd.Run(); err == nil {
		// parse out home directory
		if out := strings.TrimSpace(stdout.String()); out != "" {
			// username:password:uid:gid:gecos:home:shell
			sp := strings.SplitN(out, ":", 7)
			if len(sp) > 5 {
				if dir = sp[5]; dir != "" {
					return
				}
			}
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
