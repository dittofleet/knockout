// Package app is knockout's description of itself for go-cli-kit.
package app

import (
	clikit "github.com/dittofleet/go-cli-kit"

	"github.com/dittofleet/knockout/internal/service"
)

// Name is the install name, the repo name, and the crosstalk name the
// daemon listens on.
const Name = "knockout"

// New describes this build. The version is stamped into main at build
// time, so main passes it in.
func New(version string) clikit.App {
	return clikit.App{Owner: "dittofleet", Name: Name, Version: version, AfterUpdate: service.Restart}
}
