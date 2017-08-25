package cmd

import (
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func preRunGetClusterConfig(cmd *cobra.Command, args []string) error {
	if !cmd.Flag("config").Changed {
		fmt.Printf("config file path not given, using default config file location (%s)\n", ClusterConfigPath)
	}

	_, err := os.Stat(ClusterConfigPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("File %s does not exist!", ClusterConfigPath)
	}

	if err != nil {
		return err
	}

	if err := initClusterConfig(ClusterConfigPath); err != nil {
		return err
	}

	return nil
}

func pullKrakenContainerImage(containerImage string) (*client.Client, context.Context, error) {
	terminalSpinner.Prefix = fmt.Sprintf("Pulling image '%s' ", containerImage)
	terminalSpinner.Start()

	cli, err := getClient()
	if err != nil {
		return nil, nil, err
	}

	backgroundCtx := getContext()
	authConfig64, err := getAuthConfig64(cli, backgroundCtx)
	if err != nil {
		return nil, nil, err
	}

	if err = pullImage(cli, backgroundCtx, authConfig64); err != nil {
		return nil, nil, err
	}

	terminalSpinner.Stop()
	return cli, backgroundCtx, nil
}
