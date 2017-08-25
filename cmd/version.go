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
	"fmt"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

var KrakenMajorMinorPatch string
var KrakenType = "alpha"
var KrakenGitCommit string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display cli version",
	Long:  `Display cli version information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		semVer, err := semver.Make(KrakenMajorMinorPatch + "-" + KrakenType + "+git.sha." + KrakenGitCommit)
		if err != nil {
			return err
		}

		fmt.Println(semVer.String())
		ExitCode = 0
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
