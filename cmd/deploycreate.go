package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/renderinc/cli/pkg/client"
	"github.com/renderinc/cli/pkg/command"
	"github.com/renderinc/cli/pkg/deploy"
	"github.com/renderinc/cli/pkg/resource"
	"github.com/renderinc/cli/pkg/text"
	"github.com/renderinc/cli/pkg/tui/views"
	"github.com/renderinc/cli/pkg/types"
)

var deployCmd = &cobra.Command{
	Use:     "deploys",
	Short:   "Manage service deploys",
	GroupID: GroupCore.ID,
}

var deployCreateCmd = &cobra.Command{
	Use:   "create [serviceID]",
	Short: "Trigger a service deploy and tail logs",
	Args:  cobra.MaximumNArgs(1),
}

var InteractiveDeployCreate = func(ctx context.Context, input types.DeployInput, breadcrumb string) tea.Cmd {
	return command.AddToStackFunc(
		ctx,
		deployCreateCmd,
		breadcrumb,
		&input,
		views.NewDeployCreateView(ctx, input, func(d *client.Deploy) tea.Cmd {
			return TailResourceLogs(ctx, input.ServiceID)
		}))
}

func init() {
	deployCreateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		var input types.DeployInput
		err := command.ParseCommand(cmd, args, &input)
		if err != nil {
			return fmt.Errorf("failed to parse command: %w", err)
		}

		// if wait flag is used, default to non-interactive output
		outputFormat := command.GetFormatFromContext(cmd.Context())
		if input.Wait && outputFormat.Interactive() {
			output := command.TEXT
			cmd.SetContext(command.SetFormatInContext(cmd.Context(), &output))
		}

		nonInteractive := nonInteractiveDeployCreate(cmd, input)
		if nonInteractive {
			return nil
		}

		service, err := resource.GetResource(cmd.Context(), input.ServiceID)
		if err != nil {
			return err
		}

		InteractiveDeployCreate(cmd.Context(), input, "Create Deploy for "+resource.BreadcrumbForResource(service))
		return nil
	}

	deployCreateCmd.Flags().Bool("clear-cache", false, "Clear build cache before deploying")
	deployCreateCmd.Flags().String("commit", "", "The commit ID to deploy")
	deployCreateCmd.Flags().String("image", "", "The Docker image URL to deploy")
	deployCreateCmd.Flags().Bool("wait", false, "Wait for deploy to finish. Returns non-zero exit code if deploy fails")

	deployCmd.AddCommand(deployCreateCmd)
	rootCmd.AddCommand(deployCmd)
}

func nonInteractiveDeployCreate(cmd *cobra.Command, input types.DeployInput) bool {
	var dep *client.Deploy
	createDeploy := func() (*client.Deploy, error) {
		d, err := views.CreateDeploy(cmd.Context(), input)
		if err != nil {
			return nil, err
		}

		if input.Wait {
			_, err = fmt.Fprintf(cmd.OutOrStderr(), "Waiting for deploy %s to complete...\n\n", d.Id)
			if err != nil {
				return nil, err
			}
			dep, err = views.WaitForDeploy(cmd.Context(), input.ServiceID, d.Id)
			return dep, err
		}

		return d, err
	}

	nonInteractive, err := command.NonInteractiveWithConfirm(cmd, createDeploy, text.Deploy(input.ServiceID), views.DeployCreateConfirm(cmd.Context(), input))
	if err != nil {
		_, err = fmt.Fprintf(cmd.OutOrStderr(), err.Error()+"\n")
		os.Exit(1)
	}
	if !nonInteractive {
		return false
	}

	if input.Wait && !deploy.IsSuccessful(dep.Status) {
		os.Exit(1)
	}

	return nonInteractive
}
