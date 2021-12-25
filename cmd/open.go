package cmd

import "github.com/spf13/cobra"

var openCmd = &cobra.Command{
	Use: "open",
	Short: "Start a bridge instance within the current folder.",
	Long: `Creates a bridge instance that is open for clients to connect to and sync with.
All folders and files within the current directory will be synced with the connecting clients.
	
When a bridge has successfully opened, you will be prompted with the Session ID that can
be shared with those you wish to connect with.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
