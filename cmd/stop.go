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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bobbytrapz/autosr/options"
	"github.com/spf13/cobra"
)

func readPidAndKill() error {
	pidPath := filepath.Join(options.ConfigPath, pidFileName)
	data, err := ioutil.ReadFile(pidPath)

	if os.IsNotExist(err) {
		return errors.New("autosr is not running. (pid file not found)")
	}

	if err != nil {
		return err
	}
	defer os.Remove(pidPath)

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	fmt.Printf("autosr (%d)\n", pid)
	proc.Kill()
	_, err = proc.Wait()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Kill background autosr process",
	Long:  `Reads the pid of the background autosr from pidfile in config directory and kills the process`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := readPidAndKill(); err != nil {
			fmt.Println(err)
		}
	},
}
