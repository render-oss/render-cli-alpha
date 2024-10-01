/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/renderinc/render-cli/pkg/cfg"
	"github.com/renderinc/render-cli/pkg/client"
	lclient "github.com/renderinc/render-cli/pkg/client/logs"
	"github.com/renderinc/render-cli/pkg/command"
	"github.com/renderinc/render-cli/pkg/config"
	"github.com/renderinc/render-cli/pkg/logs"
	"github.com/renderinc/render-cli/pkg/pointers"
	"github.com/renderinc/render-cli/pkg/tui"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Return logs",
	Long:  `Return logs`,
}

var InteractiveLogs = command.Wrap(logsCmd, loadLogData, renderLogs)

type LogInput struct {
	ResourceIDs []string `cli:"resources"`
	Instance    []string `cli:"instance"`
	StartTime   *string  `cli:"start"`
	EndTime     *string  `cli:"end"`
	Text        []string `cli:"text"`
	Level       []string `cli:"level"`
	Type        []string `cli:"type"`

	Host       []string `cli:"host"`
	StatusCode []string `cli:"status-code"`
	Method     []string `cli:"method"`
	Path       []string `cli:"path"`

	Limit     int    `cli:"limit"`
	Direction string `cli:"direction"`
	Tail      bool   `cli:"tail"`
}

type LogResult struct {
	Logs       *client.Logs200Response
	LogChannel <-chan *lclient.Log
}

func (l LogInput) String() []string {
	return []string{}
}

func (l LogInput) ToParam() (*client.ListLogsParams, error) {
	now := time.Now()
	ownerID, err := config.WorkspaceID()
	if err != nil {
		return nil, fmt.Errorf("error getting workspace ID: %v", err)
	}

	return &client.ListLogsParams{
		Resource:   l.ResourceIDs,
		OwnerId:    ownerID,
		Instance:   pointers.FromArray(l.Instance),
		Limit:      pointers.From(l.Limit),
		StartTime:  command.ParseTime(now, l.StartTime),
		EndTime:    command.ParseTime(now, l.EndTime),
		Text:       pointers.FromArray(l.Text),
		Level:      pointers.FromArray(l.Level),
		Type:       pointers.FromArray(l.Type),
		Host:       pointers.FromArray(l.Host),
		StatusCode: pointers.FromArray(l.StatusCode),
		Method:     pointers.FromArray(l.Method),
		Path:       pointers.FromArray(l.Path),
		Direction:  pointers.From(mapDirection(l.Direction)),
	}, nil
}

func mapDirection(direction string) lclient.LogDirection {
	switch direction {
	case "forward":
		return lclient.Forward
	case "backward":
		return lclient.Backward
	default:
		return lclient.Backward
	}
}

func loadLogData(ctx context.Context, in LogInput) (*LogResult, error) {
	c, err := client.ClientWithAuth(&http.Client{}, cfg.GetHost(), cfg.GetAPIKey())
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}
	logRepo := logs.NewLogRepo(c)
	params, err := in.ToParam()
	if err != nil {
		return nil, fmt.Errorf("error converting input to params: %v", err)
	}

	if in.Tail {
		logChan, err := logRepo.TailLogs(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("error tailing logs: %v", err)
		}
		return &LogResult{Logs: &client.Logs200Response{}, LogChannel: logChan}, nil
	}

	logs, err := logRepo.ListLogs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error listing logs: %v", err)
	}
	return &LogResult{Logs: logs, LogChannel: nil}, nil
}

func logForm(ctx context.Context, in LogInput) *tui.FilterModel {
	form, result := command.HuhForm(logsCmd, &in)
	return tui.NewFilterModel(form.WithHeight(10), func(form *huh.Form) tea.Cmd {
		var logInput LogInput
		err := command.StructFromFormValues(result, &logInput)
		if err != nil {
			panic(err)
		}

		return command.Wrap(logsCmd, loadLogData, renderLogs)(ctx, logInput)
	})
}

func renderLogs(ctx context.Context, loadData func(LogInput) (*LogResult, error), in LogInput) (tea.Model, error) {
	loadLogs := func() (*client.Logs200Response, <-chan *lclient.Log, error) {
		result, err := loadData(in)
		if err != nil {
			return nil, nil, err
		}

		return result.Logs, result.LogChannel, nil
	}
	model := tui.NewLogModel(logForm(ctx, in), loadLogs)
	return model, nil
}

func init() {
	logsCmd.RunE = func(cmd *cobra.Command, args []string) error {
		var input LogInput
		err := command.ParseCommand(cmd, args, &input)
		if err != nil {
			return err
		}
		InteractiveLogs(cmd.Context(), input)
		return nil
	}
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringSliceP("resources", "r", []string{}, "A list of comma separated resource IDs to query")
	err := logsCmd.MarkFlagRequired("resources")
	if err != nil {
		panic(err)
	}

	logsCmd.Flags().String("start", "", "The start time of the logs to query")
	logsCmd.Flags().String("end", "", "The end time of the logs to query")
	logsCmd.Flags().StringSlice("text", []string{}, "A list of comma separated strings to search for in the logs. Only logs that contain all of the strings will be returned. Wildcards * and regular expressions are supported.")
	logsCmd.Flags().StringSlice("level", []string{}, "A list of comma separated log levels to query")
	logsCmd.Flags().StringSlice("type", []string{}, "A list of comma separated log types to query")
	logsCmd.Flags().StringSlice("instance", []string{}, "A list of comma separated instance IDs to query")
	logsCmd.Flags().StringSlice("host", []string{}, "A list of comma separated hosts to query")
	logsCmd.Flags().StringSlice("status-code", []string{}, "A list of comma separated status codes to query")
	logsCmd.Flags().StringSlice("method", []string{}, "A list of comma separated HTTP methods to query")
	logsCmd.Flags().StringSlice("path", []string{}, "A list of comma separated paths to query")
	logsCmd.Flags().Int("limit", 100, "The maximum number of logs to return")
	logsCmd.Flags().String("direction", "backward", "The direction to query the logs. Can be 'forward' or 'backward'")

	logsCmd.Flags().Bool("tail", false, "Stream new logs")
}