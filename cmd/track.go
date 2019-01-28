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
	"bytes"
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
var listFile string

const trackListFileName = "track.list"

func init() {
	rootCmd.AddCommand(trackCmd)
	trackCmd.Flags().BoolVarP(&shouldDump, "dump", "d", false, "Dump track list")
	trackCmd.Flags().StringVarP(&listFile, "list", "l", "", "Use a given file as the new track list")
}

func copyFile(from, to string) error {
	src, err := os.OpenFile(from, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("cmd.copyFile: %s", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("cmd.copyFile: %s", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("cmd.copyFile: %s", err)
	}

	return nil
}

func appendFromStdin(fn string) error {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return fmt.Errorf("cmd.appendFromStdin: nothing in stdin")
	}

	buf := &bytes.Buffer{}
	io.Copy(buf, os.Stdin)

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("cmd.appendFromStdin: %s", err)
	}
	defer f.Close()

	buf.WriteByte('\n')
	if _, err := f.Write(buf.Bytes()); err != nil {
		fmt.Println("cmd.appendFromStdin:", err)
		os.Exit(1)
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

		if err := appendFromStdin(fn); err == nil {
			// we appended data from stdin so we are done
			return
		}

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

		if listFile != "" {
			if err := copyFile(listFile, fn); err != nil {
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
			app, err = exec.LookPath("open")
			if err != nil {
				fmt.Println("error: could not find open", err)
				return
			}
			appArgs = []string{app, "-e", fn}
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

		f, err := os.OpenFile(fn, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Printf("error: opening '%s': %s\n", fn, err)
			return
		}
		// we only wanted to create the file
		f.Close()

		if err := copyFile(fn, fn+".backup"); err != nil {
			fmt.Println("error:", err)
			return
		}

		err = syscall.Exec(app, appArgs, os.Environ())

		fmt.Printf("error: running '%s' (%v): %s\n", app, appArgs, err)
		return
	},
}
