package cmd

import (
	"fmt"
	"sparkle-service/core/startupnotify"

	"github.com/spf13/cobra"
)

var (
	coreReadyNetwork string
	coreReadyAddress string
	coreReadyToken   string
)

var coreReadyCmd = &cobra.Command{
	Use:    "__core-ready",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if coreReadyNetwork == "" || coreReadyAddress == "" || coreReadyToken == "" {
			return fmt.Errorf("core ready notification requires network, address and token")
		}
		return startupnotify.Send(coreReadyNetwork, coreReadyAddress, coreReadyToken)
	},
}

func init() {
	MainCmd.AddCommand(coreReadyCmd)

	coreReadyCmd.Flags().StringVar(&coreReadyNetwork, "network", "", "ready notification network")
	coreReadyCmd.Flags().StringVar(&coreReadyAddress, "address", "", "ready notification address")
	coreReadyCmd.Flags().StringVar(&coreReadyToken, "token", "", "ready notification token")
}
