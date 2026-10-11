package main

import (
	"errors"
	"fmt"
	"os"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/selfupdate"
	"github.com/dittofleet/go-cli-kit/updatecheck"

	"github.com/dittofleet/knockout/internal/app"
	"github.com/dittofleet/knockout/internal/cmd"
)

var errUnknownCommand = errors.New("unknown command")

var version = "dev"

const usage = `Usage: knockout <command>

Shuts this Mac down when another of your Macs asks it to, through crosstalk:

  crosstalk send <this mac> knockout '{"do":"shutdown"}'
  crosstalk send <this mac> knockout '{"do":"check"}'    # "ready", or why not
  crosstalk send --all knockout '{"do":"shutdown"}'      # every other Mac

Commands:
  setup                    Listen from login, and allow shutting down
                           without a password (asks for yours once)
  update                   Download and install the latest version
  uninstall [--yes]        Stop listening, remove the sudo rule and the binary
  version                  Print the installed version
  help                     Print this help message

The daemon logs to ~/Library/Logs/knockout.log.
`

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(0)
	}

	knockout := app.New(version)
	if err := dispatch(knockout, args); err != nil {
		if errors.Is(err, errUnknownCommand) {
			fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n\n", args[0])
			fmt.Print(usage)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}

	updatecheck.MaybeCheck(knockout, args[0])
}

func dispatch(a clikit.App, args []string) error {
	switch args[0] {
	case "setup":
		return cmd.Setup()
	case "daemon":
		return cmd.Daemon()
	case "update":
		_, err := selfupdate.Run(a)
		return err
	case "uninstall":
		return cmd.Uninstall(args[1:], a)
	case "postinstall":
		return cmd.Postinstall(a)
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	default:
		return errUnknownCommand
	}
}
