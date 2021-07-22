package cmd

import (
	"github.com/spf13/cobra"
	_init "github.com/supabase/cli/internal/init"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "FIXME",
	Long:  `FIXME`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return _init.Init()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
