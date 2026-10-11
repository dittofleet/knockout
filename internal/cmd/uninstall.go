package cmd

import (
	"errors"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/uninstall"

	"github.com/dittofleet/knockout/internal/power"
	"github.com/dittofleet/knockout/internal/service"
)

const uninstallUsage = "usage: knockout uninstall [--yes]"

// Uninstall stops the daemon and removes the service, the sudo rule and
// the binary.
func Uninstall(args []string, a clikit.App) error {
	yes := len(args) == 1 && args[0] == "--yes"
	if len(args) > 0 && !yes {
		return errors.New(uninstallUsage)
	}
	return uninstall.Run(a, yes, uninstall.Plan{
		Items: []uninstall.Item{
			{Label: "Daemon", Note: "stopped, and no longer started at login", Remove: func(string) error { return service.Remove() }},
			{Label: "Sudoers", Path: power.RulePath, Note: "takes your password", Remove: power.Disallow},
		},
	})
}
