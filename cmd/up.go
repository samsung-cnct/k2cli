// Copyright © 2016 Samsung CNCT
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var upStagesList string

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:           "up [path to K2 config file]",
	Short:         "create a K2 cluster",
	SilenceErrors: false,
	SilenceUsage:  false,
	Long:          `Creates a K2 cluster described in the specified configuration yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		k2ConfigPath = os.ExpandEnv("$HOME/.kraken/config.yaml")
		if len(args) > 0 {
			k2ConfigPath = os.ExpandEnv(args[0])
		}

		_, err := os.Stat(k2ConfigPath)
		if os.IsNotExist(err) {
			return errors.New("File " + k2ConfigPath + " does not exist!")
		}

		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		initK2Config(k2ConfigPath)

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		// terminalSpinner.Prefix = "Pulling image '" + containerImage + "' "
		// terminalSpinner.Start()

		cli := getClient()

		backgroundCtx := getContext()
		pullImage(cli, backgroundCtx, getAuthConfig64(cli, backgroundCtx))


///////////////////////////////
		// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// defer cancel()

		// client, _ := NewEnvClient()
		// reader, err := cli.ContainerLogs(ctx, "container_id", types.ContainerLogsOptions{})
		// if err != nil {
		//     log.Fatal(err)
		// }
		//
		// _, err = io.Copy(os.Stdout, reader)
		// if err != nil && err != io.EOF {
		//     log.Fatal(err)
		// }

//////////////////////////////////////


		// terminalSpinner.Stop()

		// terminalSpinner.Prefix = "Bringing up cluster '" + getContainerName() + "' "
		// terminalSpinner.Start()

		command := []string{
			"ansible-playbook",
			// verbosity,
			"-i",
			"ansible/inventory/localhost",
			"ansible/up.yaml",
			"--extra-vars",
			"config_path=" + k2ConfigPath + " config_base=" + outputLocation + " kraken_action=up ",
			"--tags",
			upStagesList,
		}

		fmt.Println(command)

		ctx, cancel := getTimedContext()
		defer cancel()

		// reader, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout:true})
		// if err != nil {
		//     log.Fatal(err)
		// }
		//
		// _, err = io.Copy(os.Stdout, reader)
		// if err != nil && err != io.EOF {
		//     log.Fatal(err)
		// }

		resp, statusCode, timeout := containerAction(cli, ctx, command, k2ConfigPath)
		defer timeout()
		streamAllTheLogs(
		cli,
		resp,
		backgroundCtx,
	  )

		// terminalSpinner.Stop()
		out, err := printContainerLogs(
			cli,
			resp,
			backgroundCtx,
		)
		// fmt.Println("now printing out: ")
		// fmt.Println(string(out[:len(out)]))   this is working, it just spits out everything at the end
		// fmt.Println("now printing err: ")
		// fmt.Println(err)


		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		if len(strings.TrimSpace(logPath)) > 0 {
			writeLog(logPath, out)
		}

		if statusCode != 0 {
			fmt.Println("ERROR bringing up " + getContainerName())
			fmt.Printf("%s", out)
			clusterHelpError(Created, k2ConfigPath)
		} else {
			fmt.Println("Done.")
			if logSuccess {
				fmt.Printf("%s", out)
			}
			clusterHelp(Created, k2ConfigPath)
		}

		ExitCode = statusCode
	},
}

func init() {
	clusterCmd.AddCommand(upCmd)
	upCmd.PersistentFlags().StringVarP(
		&upStagesList,
		"stages",
		"s",
		"all",
		"comma-separated list of K2 stages to run. Run 'k2cli help topic stages' for more info.")
}
