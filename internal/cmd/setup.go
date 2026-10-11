package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/postinstall"

	"github.com/dittofleet/knockout/internal/power"
	"github.com/dittofleet/knockout/internal/service"
)

// Postinstall is the install script's first-time setup.
func Postinstall(a clikit.App) error {
	return postinstall.Run(a, Setup)
}

// Setup starts the daemon at login and allows it to shut down without a
// password. Run again, it restarts the daemon on this binary, and only asks
// for the password if the rule is gone.
func Setup() error {
	crosstalk, err := exec.LookPath("crosstalk")
	if err != nil {
		return errors.New("crosstalk is not installed, see https://github.com/dittofleet/crosstalk")
	}
	binary, err := clikit.Executable()
	if err != nil {
		return err
	}
	if err := service.Install(binary, filepath.Dir(crosstalk)+":/usr/bin:/bin:/usr/sbin:/sbin"); err != nil {
		return err
	}
	if err := power.Allow(); err != nil {
		return err
	}
	fmt.Printf("Ready. To shut this Mac down from another one:\n\n  crosstalk send %s knockout '{\"do\":\"shutdown\"}'\n", thisMac(crosstalk))
	return nil
}

// thisMac is this Mac's name on crosstalk.
func thisMac(crosstalk string) string {
	out, _ := exec.Command(crosstalk, "devices", "--json").Output()
	var list struct {
		Devices []struct {
			Name string `json:"name"`
			Self bool   `json:"self"`
		} `json:"devices"`
	}
	json.Unmarshal(out, &list)
	for _, d := range list.Devices {
		if d.Self {
			return d.Name
		}
	}
	return "<this mac>"
}
