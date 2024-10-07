package cmd

import (
	"context"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
	"github.com/renderinc/render-cli/pkg/client"
	"github.com/renderinc/render-cli/pkg/command"
	"github.com/renderinc/render-cli/pkg/environment"
	"github.com/renderinc/render-cli/pkg/postgres"
	"github.com/renderinc/render-cli/pkg/project"
	"github.com/renderinc/render-cli/pkg/resource"
	"github.com/renderinc/render-cli/pkg/service"
	"github.com/renderinc/render-cli/pkg/tui"
	"github.com/renderinc/render-cli/pkg/types"
	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List and manage services",
}

var InteractiveServices = command.Wrap(servicesCmd, loadResourceData, renderResources)

func loadResourceData(ctx context.Context, _ ListResourceInput) ([]resource.Resource, error) {
	resourceService, err := newResourceService()
	if err != nil {
		return nil, err
	}
	return resourceService.ListResources(ctx)
}

type ListResourceInput struct{}

func (l ListResourceInput) String() []string {
	return []string{}
}

func renderResources(ctx context.Context, loadData func(input ListResourceInput) ([]resource.Resource, error), in ListResourceInput) (tea.Model, error) {
	columns := []btable.Column{
		btable.NewColumn("ID", "ID", 25).WithFiltered(true),
		btable.NewColumn("Type", "Type", 12).WithFiltered(true),
		btable.NewColumn("Project", "Project", 15).WithFiltered(true),
		btable.NewColumn("Environment", "Environment", 20).WithFiltered(true),
		btable.NewColumn("Name", "Name", 40).WithFiltered(true),
	}

	rows, err := loadServiceRows(loadData, in)
	if err != nil {
		return nil, err
	}

	onSelect := func(data []btable.Row) tea.Cmd {
		if len(data) == 0 || len(data) > 1 {
			return nil
		}

		r, ok := data[0].Data["resource"].(resource.Resource)
		if !ok {
			return nil
		}

		return selectResource(ctx)(r)
	}

	reInitFunc := func(tableModel *tui.NewTable) tea.Cmd {
		return func() tea.Msg {
			rows, err := loadServiceRows(loadData, in)
			if err != nil {
				return tui.ErrorMsg{Err: err}
			}
			tableModel.UpdateRows(rows)
			return nil
		}
	}

	customOptions := []tui.CustomOption{
		{
			Key:   "w",
			Title: "Change Workspace",
			Function: func(row btable.Row) tea.Cmd {
				return InteractiveWorkspace(ctx, ListWorkspaceInput{})
			},
		},
	}

	t := tui.NewNewTable(
		columns,
		rows,
		onSelect,
		tui.WithCustomOptions(customOptions),
		tui.WithOnReInit(reInitFunc),
	)

	return t, nil
}

func loadServiceRows(loadData func(input ListResourceInput) ([]resource.Resource, error), in ListResourceInput) ([]btable.Row, error) {
	resources, err := loadData(in)
	if err != nil {
		return nil, err
	}

	var rows []btable.Row
	for _, r := range resources {
		rows = append(rows, btable.NewRow(btable.RowData{
			"ID":          r.ID(),
			"Type":        r.Type(),
			"Project":     r.ProjectName(),
			"Environment": r.EnvironmentName(),
			"Name":        r.Name(),
			"resource":    r, // this will be hidden in the UI, but will be used to get the resource when selected
		}))
	}
	return rows, nil
}

func optionallyAddCommand(commands []PaletteCommand, command PaletteCommand, allowedTypes []string, resource resource.Resource) []PaletteCommand {
	if len(allowedTypes) == 0 {
		return append(commands, command)
	}

	for _, allowedType := range allowedTypes {
		if resource.Type() == allowedType {
			return append(commands, command)
		}
	}

	return commands
}

func selectResource(ctx context.Context) func(resource.Resource) tea.Cmd {
	return func(r resource.Resource) tea.Cmd {

		type commandWithAllowedTypes struct {
			command      PaletteCommand
			allowedTypes []string
		}

		var commands []PaletteCommand
		commandWithTypes := []commandWithAllowedTypes{
			{
				command: PaletteCommand{
					Name:        "logs",
					Description: "View resource logs",
					Action: func(ctx context.Context, args []string) tea.Cmd {
						return InteractiveLogs(ctx, LogInput{
							ResourceIDs: []string{r.ID()},
						})
					},
				},
			},
			{
				command: PaletteCommand{
					Name:        "restart",
					Description: "Restart the service",
					Action: func(ctx context.Context, args []string) tea.Cmd {
						return InteractiveRestart(ctx, RestartInput{ResourceID: r.ID()})
					},
				},
			},
			{
				command: PaletteCommand{
					Name:        "psql",
					Description: "Connect to the PostgreSQL database",
					Action: func(ctx context.Context, args []string) tea.Cmd {
						return InteractivePSQL(ctx, PSQLInput{PostgresID: r.ID()})
					},
				},
				allowedTypes: []string{postgres.PostgresType},
			},
			{
				command: PaletteCommand{
					Name:        "deploy",
					Description: "Deploy the service",
					Action: func(ctx context.Context, args []string) tea.Cmd {
						return InteractiveDeploy(ctx, types.DeployInput{ServiceID: r.ID()})
					},
				},
				allowedTypes: service.Types,
			},
			{
				command: PaletteCommand{
					Name:        "ssh",
					Description: "SSH into the service",
					Action: func(ctx context.Context, args []string) tea.Cmd {
						return InteractiveSSH(ctx, SSHInput{ServiceID: r.ID()})
					},
				},
				allowedTypes: []string{
					service.WebServiceResourceType, service.PrivateServiceResourceType,
					service.BackgroundWorkerResourceType,
				},
			},
		}

		for _, c := range commandWithTypes {
			commands = optionallyAddCommand(commands, c.command, c.allowedTypes, r)
		}

		return InteractiveCommandPalette(ctx, PaletteCommandInput{
			Commands: commands,
		})
	}
}

func newResourceService() (*resource.Service, error) {
	httpClient := http.DefaultClient
	host := os.Getenv("RENDER_HOST")
	apiKey := os.Getenv("RENDER_API_KEY")

	c, err := client.ClientWithAuth(httpClient, host, apiKey)
	if err != nil {
		return nil, err
	}

	serviceRepo := service.NewRepo(c)
	environmentRepo := environment.NewRepo(c)
	projectRepo := project.NewRepo(c)
	postgresRepo := postgres.NewRepo(c)

	serviceService := service.NewService(serviceRepo, environmentRepo, projectRepo)
	postgresService := postgres.NewService(postgresRepo, environmentRepo, projectRepo)

	resourceService := resource.NewResourceService(
		serviceService,
		postgresService,
		environmentRepo,
		projectRepo,
	)

	return resourceService, nil
}

func init() {
	rootCmd.AddCommand(servicesCmd)

	servicesCmd.RunE = func(cmd *cobra.Command, args []string) error {
		InteractiveServices(cmd.Context(), ListResourceInput{})
		return nil
	}
}
