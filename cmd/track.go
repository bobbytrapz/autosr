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

package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/bobbytrapz/autosr/options"
	"github.com/spf13/cobra"
)

var shouldDump bool

const trackListFileName = "track.list"

func init() {
	rootCmd.AddCommand(trackCmd)
	trackCmd.Flags().BoolVarP(&shouldDump, "dump", "d", false, "Dump track list")
}

func copyTo(path string) error {
	fn := filepath.Join(options.ConfigPath, trackListFileName)

	src, err := os.OpenFile(fn, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("cmd.copyTo: %s", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("cmd.copyTo: %s", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("cmd.copyTo: %s", err)
	}

	return nil
}

var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Allows you to provide a list of urls to check for streams",
	Long: `The files track.list and track.list.backup can be found in the config directory.
When you change this file the tracked targets are updated right away.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fn := filepath.Join(options.ConfigPath, trackListFileName)
		if shouldDump {
			f, err := os.Open(fn)
			if err != nil {
				fmt.Println("error:", err)
				return
			}
			defer f.Close()

			if _, err := io.Copy(os.Stdout, f); err != nil {
				fmt.Println("error:", err)
				return
			}

			return
		}

		var err error
		var app string
		var appArgs []string
		switch runtime.GOOS {
		case "darwin":
			editor := os.Getenv("EDITOR")
			if editor == "" {
				app = "open"
				appArgs = []string{"open", "-a", fn}
			} else {
				app, err = exec.LookPath(editor)
				if err != nil {
					fmt.Println("error: could not find", editor, err)
					return
				}
				appArgs = []string{app, fn}
			}
		case "windows":
			sys := os.Getenv("SYSTEM32")
			if sys != "" {
				sys = `C:\WINDOWS\System32`
			}
			app = filepath.Join(sys, `Notepad.exe`)
			appArgs = []string{app, "/W", fn}
		default:
			// assume unix system
			editor := os.Getenv("EDITOR")
			if editor == "" {
				fmt.Println("You need to set the $EDITOR flag.", err)
				return
			}
			app, err = exec.LookPath(editor)
			if err != nil {
				fmt.Println("error: could not find", editor, err)
				return
			}
			appArgs = []string{app, fn}
		}

		if err := os.MkdirAll(options.ConfigPath, 0700); err != nil {
			fmt.Println("error:", err)
			return
		}

		if err := copyTo(fn + ".backup"); err != nil {
			fmt.Println("error:", err)
			return
		}

		f, err := os.OpenFile(fn, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		// we only wanted to create the file
		f.Close()

		err = syscall.Exec(app, appArgs, os.Environ())

		fmt.Println("error:", err)
		return
	},
}
