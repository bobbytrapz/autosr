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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/bobbytrapz/autosr/options"
	"github.com/spf13/cobra"
)

var optionsEditor string

func init() {
	rootCmd.AddCommand(optionsCmd)
	optionsCmd.LocalFlags().StringVarP(&optionsEditor, "editor", "e", os.Getenv("EDITOR"), "Command to use for editing.")
}

var optionsCmd = &cobra.Command{
	Use:   "options",
	Short: "Allows you to edit the autosr config file",
	Run: func(cmd *cobra.Command, args []string) {
		fn := filepath.Join(options.ConfigPath, options.Filename+"."+options.Format)

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
		default:
			// assume unix system
			app, err = exec.LookPath(optionsEditor)
			if err != nil {
				fmt.Println("error: could not find", optionsEditor, err)
				return
			}
			appArgs = []string{app, fn}
		}

		if err := os.MkdirAll(options.ConfigPath, 0700); err != nil {
			fmt.Println("error:", err)
			return
		}

		f, err := os.OpenFile(fn, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		f.Close()

		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd.exe", "/C", "start", "/b", "Notepad", fn)
			if err = cmd.Start(); err == nil {
				return
			}
		} else {
			err = syscall.Exec(app, appArgs, os.Environ())
		}

		fmt.Println("error:", err)
		return
	},
}
