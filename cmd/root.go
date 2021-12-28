package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Bridge is a simple peer-to-peer fs sync CLI tool.",
	Long: `Bridge aims to be fast and simple approach to syncing content within a folder.

Simply start up a bridge instance and share the session with whomever you wish to sync your folder with. 
All they need to do is to use the bridge instance and `,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
