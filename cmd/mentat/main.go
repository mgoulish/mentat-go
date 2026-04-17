package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mgoulish/mentat/internal/cli"
	"github.com/mgoulish/mentat/internal/config"
	"github.com/mgoulish/mentat/internal/connectivity"
	"github.com/mgoulish/mentat/internal/debug"
	"github.com/mgoulish/mentat/internal/new"
	"github.com/mgoulish/mentat/internal/parser"
)

func main() {
	rootPtr := flag.String("root", "", "root dir for the network run (contains site dirs like wynford/, dorval/)")
	info := flag.Bool("info", false, "Print info messages")
	dbg := flag.Bool("debug", false, "Print debug messages")
	script := flag.String("script", "", "Run commands from this script file (not yet implemented)")

	flag.Parse()

	if *rootPtr == "" {
		fmt.Println("Error: --root is required")
		flag.Usage()
		os.Exit(1)
	}

	// Clean and validate root path
	root := filepath.Clean(*rootPtr)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		fmt.Printf("Error: root directory does not exist: %s\n", root)
		os.Exit(1)
	}

	debug.SetInfo(*info)
	debug.SetDebug(*dbg)

	fmt.Printf("Mentat loading data from: %s\n\n", root)

	mentat := new.NewMentat(root)

	config.ReadNetwork(mentat)
	parser.ReadEvents(mentat)
	connectivity.ReadConnectivityEvents(mentat)

	debug.Info(fmt.Sprintf("mentat now has %d total events", len(mentat.Events)))

	// Start CLI
	c := cli.NewMentatCLI(mentat)
	c.SetRoot(root)   // ← fixed: use setter instead of direct field access

	// Handle script if provided
	if *script != "" {
		fmt.Printf("Warning: script support not yet implemented. Ignoring --script %s\n", *script)
	}

	if err := c.Run(); err != nil {
		fmt.Printf("CLI error: %v\n", err)
		os.Exit(1)
	}
}
