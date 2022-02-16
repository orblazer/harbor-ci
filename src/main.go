package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/orblazer/harbor-cli/api"
)

// flags
var (
	username string
	password string
	url      string

	testCmd = flag.NewFlagSet("test", flag.ExitOnError)
)

var subcommands = map[string]*flag.FlagSet{
	testCmd.Name(): testCmd,
}

var apiClient *api.Client

func main() {
	setupCommonFlags()

	// Find subcommand
	cmd := subcommands[os.Args[1]]
	if cmd == nil {
		log.Fatalf("[ERROR] unknown subcommand '%s', see help for more details.", os.Args[1])
	}

	// Parse flags
	cmd.Parse(os.Args[2:])

	// Require credentials
	if username == "" {
		log.Fatal("[ERROR] missing argument: -username")
	}
	if password == "" {
		log.Fatal("[ERROR] missing argument: -password")
	}
	if url == "" {
		log.Fatal("[ERROR] missing argument: -url")
	}

	// Fix url
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	// Create api client
	apiClient = api.NewClient(url, username, password)

	switch cmd.Name() {
	case "test":
		log.Println("Hello World!")
	}
}

func setupCommonFlags() {
	for _, fs := range subcommands {
		fs.StringVar(&username, "username", "", "The username")
		fs.StringVar(&password, "password", "", "The password")
		fs.StringVar(&url, "url", "", "The api base url")
	}
}
