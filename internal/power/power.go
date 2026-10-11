// Package power shuts this Mac down, through a sudo rule that lets the
// user run exactly that one command without a password.
//
// shutdown(8) has launchd stop every process with SIGTERM, and SIGKILL the
// ones that are still there after a while, then syncs and unmounts the
// disks. Apps get no chance to ask whether to save first, so nothing can
// hold the shutdown up.
package power

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// RulePath is the sudo rule's file.
const RulePath = "/etc/sudoers.d/knockout"

var shutdown = []string{"/sbin/shutdown", "-h", "now"}

// errNotAllowed is the reply to a Mac asking for a shutdown before setup
// has run.
var errNotAllowed = errors.New("not allowed to shut down without a password, run `knockout setup` on it")

// Ready reports whether Shutdown would go through. sudo -l alone is not
// enough: it can list a command that would still take a password, so the
// rule has to be there too.
func Ready() error {
	if _, err := os.Stat(RulePath); err != nil {
		return errNotAllowed
	}
	if exec.Command("sudo", append([]string{"-n", "-l"}, shutdown...)...).Run() != nil {
		return errNotAllowed
	}
	return nil
}

// Shutdown shuts this Mac down. It returns once the shutdown has started.
func Shutdown() error {
	out, err := exec.Command("sudo", append([]string{"-n"}, shutdown...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(shutdown, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Allow installs the sudo rule, asking for the user's password, unless
// shutting down is allowed already.
func Allow() error {
	if Ready() == nil {
		return nil
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "knockout-sudoers")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	rule := fmt.Sprintf("# Lets knockout shut this Mac down without a password.\n%s ALL=(root) NOPASSWD: %s\n",
		u.Username, strings.Join(shutdown, " "))
	if _, err := tmp.WriteString(rule); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// A broken file in sudoers.d breaks sudo altogether, so it is checked
	// before it goes in.
	if out, err := exec.Command("/usr/sbin/visudo", "-cf", tmp.Name()).CombinedOutput(); err != nil {
		return fmt.Errorf("the sudo rule does not check out: %s", strings.TrimSpace(string(out)))
	}
	fmt.Println("Allowing shutdown without a password takes yours, once.")
	if err := sudo("install", "-m", "0440", "-o", "root", "-g", "wheel", tmp.Name(), RulePath); err != nil {
		return err
	}
	if Ready() != nil {
		return fmt.Errorf("sudo still wants a password for shutdown, check that /etc/sudoers includes %s", filepath.Dir(RulePath))
	}
	return nil
}

// Disallow removes the sudo rule, asking for the user's password. With no
// rule it does nothing.
func Disallow(string) error {
	if _, err := os.Stat(RulePath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return sudo("rm", "-f", RulePath)
}

// sudo runs a command as root on the terminal, so sudo can ask for the
// password there.
func sudo(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo %s: %w", args[0], err)
	}
	return nil
}
