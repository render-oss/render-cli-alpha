package validate

import (
	"fmt"

	"github.com/renderinc/render-cli/pkg/config"
)

// WorkspaceMatches gets the workspace from the config and validates that it matches the provided input. If the
// workspace is not set, no error is returned
func WorkspaceMatches(workspaceID string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Workspace != "" && cfg.Workspace != workspaceID {
		return fmt.Errorf("resource in workspace %s does not match the workspace in the current workspace context %s. Run `render workspace` to change contexts", workspaceID, cfg.Workspace)
	}
	return nil
}