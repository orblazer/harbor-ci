package commands

import (
	"fmt"

	"github.com/orblazer/harbor-cli/build"
)

func Version() {
	fmt.Printf("Version: %s\n", build.Version)
	fmt.Printf("Build time: %s\n", build.Time)
	fmt.Printf("Build revision: %s\n", build.Revision)
}
