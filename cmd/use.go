package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use: "use SessionID",
	Short: "Connect to a bridge and start file sync.",
	Long: `Uses the provided session id token to connect with a bridge instance and when a
connection has be successful file sync will automatically start within the current directory.

SessionID - Provided ID of the bridge instance that you intent to connect with.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expects exactly 1 session id but got %v", len(args))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
