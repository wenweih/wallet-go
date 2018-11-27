package main

import (
	"github.com/spf13/cobra"
	"wallet-transition/pkg/configure"
)

var rootCmd = &cobra.Command{
	Use:   "wallet-transition-tool",
	Short: "Commandline to for anbi exchange wallet module",
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		configure.Sugar.Fatal("Command execute error: ", err.Error())
	}
}

var migrateWallet = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate wallet from blockchain node",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func main() {
	execute()
}

func init() {
	rootCmd.AddCommand(migrateWallet)
}
