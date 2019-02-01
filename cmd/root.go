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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/bobbytrapz/autosr/dashboard"
	"github.com/bobbytrapz/autosr/ipc"
	"github.com/bobbytrapz/autosr/options"
	// use showroom module
	_ "github.com/bobbytrapz/autosr/showroom"
	"github.com/bobbytrapz/autosr/track"
	"github.com/spf13/cobra"
)

const (
	pidFileName = ".autosr-pid"
)

const backgroundEnvKey = "autosr_is_now_running_in_the_background"

func isRunningInBackground() bool {
	// check pid file
	pidPath := filepath.Join(options.ConfigPath, pidFileName)
	_, err := os.Stat(pidPath)
	return err == nil
}

func runSelfInBackground() (*exec.Cmd, error) {
	// get the path of our executable
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// build a command with a modified environment
	cmd := exec.Command(exePath)
	env := os.Environ()
	env = append(env, backgroundEnvKey+"=1")
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("We could not start in the background: %s", err)
	}

	// write a pid file
	runinfo := fmt.Sprintf("%d", cmd.Process.Pid)
	pidPath := filepath.Join(options.ConfigPath, pidFileName)
	if err := ioutil.WriteFile(pidPath, []byte(runinfo), 0644); err != nil {
		panic(err)
	}

	return cmd, nil
}

var rootCmd = &cobra.Command{
	Use:   "autosr",
	Short: "autosr: Automated Scheduled Recordings",
	Long: `autosr: Automated Scheduled Recordings
autosr tracks users and records their livestreams when they start.
autosr was written by Bobby (@pibisubukebe) so that he never misses 齊藤京子.

If you have a comment or suggestion please contact @pibisubukebe on Twitter.
For help or to report a bug visit https://github.com/bobbytrapz/autosr.

Dashboard controls:
	q - Quit dashboard. Use 'autosr stop' to actually stop autosr
	r - Check if any streams are on right away
	o - Open link to stream in web browser
	right-click - Open link to stream in web browser

This program comes with ABSOLUTELY NO WARRANTY;
This is free software, and you are welcome to redistribute it under certain conditions.
Details can be found at https://github.com/bobbytrapz/autosr/LICENSE.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := options.AreValid(); !ok {
			fmt.Println(err)
			os.Exit(1)
		}

		if !shouldRunInForeground && os.Getenv(backgroundEnvKey) == "" {
			if isRunningInBackground() {
				if shouldNotStartDashboard {
					return
				}
				dashboard.Run(shouldColorLogo)
				return
			}

			_, err := runSelfInBackground()
			if err != nil {
				panic(err)
			}

			if shouldNotStartDashboard {
				return
			}

			<-time.After(1 * time.Second)
			dashboard.Run(shouldColorLogo)
			return
		}
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// start ipc
		ipc.Start(ctx)

		// start tracking
		if err := track.Start(ctx); err != nil {
			panic(err)
		}

		// wait for all tracking related tasks to complete
		defer func() {
			track.Wait()
			fmt.Println("autosr: done")
		}()

		// handle interrupt
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)

		for {
			select {
			case <-sig:
				signal.Stop(sig)
				fmt.Println("autosr: caught signal")
				cancel()
			case <-ctx.Done():
				fmt.Println("autosr: finishing...")
				return
			}
		}
	},
}

var shouldRunInForeground = false
var shouldNotStartDashboard = false
var shouldColorLogo = false

func init() {
	rootCmd.Flags().BoolVarP(&shouldRunInForeground, "foreground", "f", false, "Run autosr in the foreground")
	rootCmd.Flags().BoolVarP(&shouldNotStartDashboard, "no-dashboard", "d", false, "Do not start the dashboard")
	rootCmd.Flags().BoolVar(&shouldColorLogo, "color", false, "Use the colorful logo")
}

// Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
