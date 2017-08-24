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
	"github.com/spf13/cobra"
	"os"
)

var userName string
var password string
var configForced bool


// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "K2 cluster actions",
	Long:  `Commands that perform actions on a K2 cluster described by a provided yaml config`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
		ExitCode = 1
	},
}

func init() {
	RootCmd.AddCommand(clusterCmd)

	clusterCmd.PersistentFlags().StringVarP(
		&k2ConfigPath,
		"config",
		"c",
		os.ExpandEnv("$HOME/.kraken/config.yaml"),
		"Path to the kraken cluster config")
	clusterCmd.PersistentFlags().BoolVarP(
		&configForced,
		"force",
		"f",
		false,
		"true if operation should be proceed even if config is deprecated (default false)")
	clusterCmd.PersistentFlags().StringVarP(
		&password,
		"password",
		"p",
		"",
		"registry password")
	clusterCmd.PersistentFlags().StringVarP(
		&userName,
		"user",
		"u",
		"",
		"registry user name")

}
