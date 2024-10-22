package cmd

import (
	"fmt"

	"github.com/renderinc/render-cli/pkg/client"
	"github.com/renderinc/render-cli/pkg/config"
	"github.com/renderinc/render-cli/pkg/owner"
	"github.com/spf13/cobra"
)

var workspaceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently selected workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.NewDefaultClient()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ownerRepo := owner.NewRepo(c)
		workspace, err := config.WorkspaceID()
		if err != nil {
			return err
		}

		owner, err := ownerRepo.RetrieveOwner(cmd.Context(), workspace)
		if err != nil {
			return fmt.Errorf("failed to list owners: %w", err)
		}

		fmt.Printf("Active Workspace: %s (%s)\n", owner.Name, owner.Id)
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceCurrentCmd)
}
