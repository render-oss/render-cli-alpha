package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/renderinc/render-cli/pkg/command"
	"github.com/renderinc/render-cli/pkg/tui"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [resourceID]",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
}

var InteractiveRestart = command.Wrap(restartCmd, restartResource, renderRestart)

type RestartInput struct {
	ResourceID string
}

func (r RestartInput) String() []string {
	return []string{r.ResourceID}
}

func restartResource(ctx context.Context, input RestartInput) (string, error) {
	resourceService, err := newResourceService()
	if err != nil {
		return "", fmt.Errorf("failed to create resource service: %w", err)
	}

	err = resourceService.RestartResource(ctx, input.ResourceID)
	if err != nil {
		return "", fmt.Errorf("failed to restart resource: %w", err)
	}

	return fmt.Sprintf("%s restarted successfully", input.ResourceID), nil
}

func renderRestart(_ context.Context, loadData func(RestartInput) (string, error), in RestartInput) (tea.Model, error) {
	return tui.NewSimpleModel(func() (string, error) {
		return loadData(in)
	}), nil
}

func init() {
	restartCmd.RunE = func(cmd *cobra.Command, args []string) error {
		var input RestartInput
		if len(args) == 1 {
			input.ResourceID = args[0]
		} else {
			err := command.ParseCommand(cmd, args, &input)
			if err != nil {
				return err
			}
		}

		InteractiveRestart(cmd.Context(), input)
		return nil
	}

	rootCmd.AddCommand(restartCmd)
}
