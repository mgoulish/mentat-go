package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/mgoulish/mentat/internal/cli"
    "github.com/mgoulish/mentat/internal/config"
    "github.com/mgoulish/mentat/internal/connectivity"
    "github.com/mgoulish/mentat/internal/debug"
    "github.com/mgoulish/mentat/internal/new"
    "github.com/mgoulish/mentat/internal/parser"
)

func main() {
    rootPtr := flag.String("root", "", "root dir for the network run (contains site dirs)")
    info := flag.Bool("info", false, "Print info messages")
    dbg := flag.Bool("debug", false, "Print debug messages")
    script := flag.String("script", "", "Run commands from this script file")

    flag.Parse()

    if *rootPtr == "" {
        fmt.Println("Error: --root is required")
        flag.Usage()
        os.Exit(1)
    }

    debug.SetInfo(*info)
    debug.SetDebug(*dbg)

    mentat := new.NewMentat(*rootPtr)

    config.ReadNetwork(mentat)
    parser.ReadEvents(mentat)
    connectivity.ReadConnectivityEvents(mentat)

    debug.Info(fmt.Sprintf("mentat now has %d total events", len(mentat.Events)))

    // Start CLI
    c := cli.NewMentatCLI(mentat)

    if *script != "" {
        if err := c.RunScript(*script); err != nil {
            fmt.Printf("Error running script: %v\n", err)
        }
    }

    if err := c.Run(); err != nil {
        fmt.Printf("CLI error: %v\n", err)
        os.Exit(1)
    }
}
