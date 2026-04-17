package cli

import (
    "bufio"
    "fmt"
    "os"
    "sort"
    "strconv"
    "strings"
    "time"

    "github.com/mgoulish/mentat/internal/connectivity"
    "github.com/mgoulish/mentat/internal/new"
    "github.com/mgoulish/mentat/internal/utils"
)

type MentatCLI struct {
    mentat *new.Mentat
}

func NewMentatCLI(m *new.Mentat) *MentatCLI {
    return &MentatCLI{mentat: m}
}

func (c *MentatCLI) Run() error {
    fmt.Println("Mentat CLI started (Go port). Type 'help' for commands.")
    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Print("(mentat) ")
        if !scanner.Scan() {
            break
        }
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        c.execute(line)

        if line == "quit" || line == "q" {
            fmt.Println("quitting...")
            break
        }
    }
    return nil
}

func (c *MentatCLI) RunScript(path string) error {
    fmt.Printf("Running script %s (not implemented yet)\n", path)
    return nil
}

func (c *MentatCLI) execute(line string) {
    parts := strings.Fields(line)
    if len(parts) == 0 {
        return
    }
    cmd := parts[0]
    args := parts[1:]

    switch cmd {
    case "overview":
        c.doOverview()
    case "sites":
        c.doSites()
    case "state":
        c.doState(args)
    case "debug-events":
        c.doDebugEvents()
    case "events":
        c.doEvents(args)
    case "debug-routers":
        c.doDebugRouters()
    case "range":
        c.doRange(args)
    case "errors":
        c.doErrors(args)
    case "help":
        c.doHelp()
    case "quit", "q":
        // handled above
    default:
        fmt.Printf("Unknown command: %s\n", cmd)
    }
}

func (c *MentatCLI) doHelp() {
    fmt.Println("\nAvailable commands:")
    fmt.Println("  overview               Show summary of loaded data")
    fmt.Println("  sites                  List sites and routers")
    fmt.Println("  events [n]             Show first n events (default 20)")
    fmt.Println("  range <start> <end>    Show events in ID range")
    fmt.Println("  errors [minutes]       Show error clumps (default 5 minutes)")
    fmt.Println("  state [timestamp]      Show connectivity state at a time")
    fmt.Println("                         (example: state 2025-09-16 04:30:00)")
    fmt.Println("                         (no timestamp = first event time)")
    fmt.Println("  help                   Show this help")
    fmt.Println("  quit / q               Exit")
    fmt.Println()
}

/*
func (c *MentatCLI) doHelp() {
    fmt.Println("\nAvailable commands:")
    fmt.Println("  overview               Show summary of loaded data")
    fmt.Println("  sites                  List sites and routers")
    fmt.Println("  events [n]             Show first n events (default 20)")
    fmt.Println("  range <start> <end>    Show events in ID range")
    fmt.Println("  errors [minutes]       Show error clumps (default 5 minutes)")
    fmt.Println("  help                   Show this help")
    fmt.Println("  quit / q               Exit")
    fmt.Println()
}
*/


func (c *MentatCLI) doOverview() {
    fmt.Printf("\n=== Data Overview ===\n")
    fmt.Printf("Total log events:       %d\n", len(c.mentat.Events))
    fmt.Printf("Connectivity events:    %d\n", len(c.mentat.ConnectivityEvents))
    fmt.Printf("Sites:                  %d\n\n", len(c.mentat.Sites))

    counts := make(map[string]int)
    for _, ev := range c.mentat.ConnectivityEvents {
        if t, ok := ev["type"].(string); ok {
            counts[t]++
        }
    }

    fmt.Printf("Configured connectors:  %d\n", counts["configured_connector"])
    fmt.Printf("Configured listeners:   %d\n", counts["configured_listener"])
    fmt.Printf("HTTP listeners:         %d\n", counts["http_listener"])
    fmt.Printf("Client listeners:       %d\n", counts["client_listener"])
    fmt.Printf("TCP listeners:          %d\n", counts["tcp_listener"])
    fmt.Printf("TCP connectors:         %d\n", counts["tcp_connector"])
    fmt.Printf("Server listening:       %d\n", counts["server_listening"])
    fmt.Printf("Errors:                 %d\n", counts["error"])
    fmt.Println()
}
func (c *MentatCLI) doSites() {
    for _, s := range c.mentat.Sites {
        fmt.Printf("site: %s   routers: %d\n", s.Name, len(s.Routers))
        for _, r := range s.Routers {
            fmt.Printf("  → router: %s (nickname: %s)\n", r.Name, r.Nickname)
        }
    }
}

func (c *MentatCLI) doEvents(args []string) {
    n := 20
    if len(args) > 0 {
        if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
            n = v
        }
    }
    c.showEvents(1, n)
}

func (c *MentatCLI) doRange(args []string) {
    if len(args) < 2 {
        fmt.Println("Usage: range <start_id> <end_id>")
        return
    }
    start, _ := strconv.Atoi(args[0])
    end, _ := strconv.Atoi(args[1])
    c.showEvents(start, end)
}

func (c *MentatCLI) doErrors(args []string) {
    minutes := 5
    if len(args) > 0 {
        if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
            minutes = v
        }
    }

    var errorEvents []new.Event
    for _, ev := range c.mentat.ConnectivityEvents {
        if t, ok := ev["type"].(string); ok && t == "error" {
            if msg, ok := ev["message"].(string); ok {
                ts := ev["timestamp"].(string)
                micros, _ := utils.StringToMicrosecondsSinceEpoch(ts + " +0000")
                errorEvents = append(errorEvents, new.Event{
                    Timestamp: ts,
                    Micros:    micros,
                    Line:      msg,
                    Router:    ev["router"].(string),
                })
            }
        }
    }

    if len(errorEvents) == 0 {
        fmt.Println("No errors found.")
        return
    }

    fmt.Printf("\n=== Errors (clumped within %d minutes) ===\n\n", minutes)
    c.showClumpedErrors(errorEvents, minutes)
}

func (c *MentatCLI) doState(args []string) {
    // If no timestamp given, use a reasonable default (first event time)
    if len(args) == 0 {
        if len(c.mentat.ConnectivityEvents) == 0 {
            fmt.Println("No connectivity events loaded yet. Try loading a log first.")
            return
        }
        // Default to time of first event
        firstEv := c.mentat.ConnectivityEvents[0]
        if ts, ok := firstEv["timestamp"].(string); ok {
            t, _ := time.Parse("2006-01-02 15:04:05", ts) // rough parse
            state := connectivity.StateAt(c.mentat.ConnectivityEvents, t)
            fmt.Println(state.String())
        }
        return
    }

    // Parse user-provided timestamp
    timeStr := strings.Join(args, " ")
    t, err := parseTimeFlexible(timeStr)
    if err == nil {
        t = t.UTC()   // ← Force UTC to match event storage
    }
    if err != nil {
        fmt.Printf("Error parsing time '%s': %v\n", timeStr, err)
        fmt.Println("Examples: 2025-09-08 22:10:00   or   2025-09-16 04:17:20")
        return
    }

    // Ensure events are loaded
    if len(c.mentat.ConnectivityEvents) == 0 {
        fmt.Println("No connectivity events loaded. The state command needs them.")
        return
    }

    state := connectivity.StateAt(c.mentat.ConnectivityEvents, t)
    fmt.Println(state.String())
}

func (c *MentatCLI) doDebugRouters() {
    fmt.Println("Searching for inter-router connector events...")

    count := 0
    for _, ev := range c.mentat.ConnectivityEvents {
        if typ, ok := ev["type"].(string); ok && typ == "flow_connector" {
            host := ""
            if h, ok := ev["dest_host"]; ok {
                host = fmt.Sprintf("%v", h)
            }
            ts := ev["timestamp"]
            fmt.Printf("%s | flow_connector → dest_host = %s\n", ts, host)
            count++
        }
    }

    if count == 0 {
        fmt.Println("No flow_connector events found at all.")
    } else {
        fmt.Printf("Found %d flow_connector events total.\n", count)
    }
}

func (c *MentatCLI) doDebugEvents() {
    fmt.Printf("Total ConnectivityEvents: %d\n", len(c.mentat.ConnectivityEvents))
    if len(c.mentat.ConnectivityEvents) == 0 {
        fmt.Println("No events loaded.")
        return
    }

    fmt.Println("\nFirst 10 event types:")
    for i := 0; i < 10 && i < len(c.mentat.ConnectivityEvents); i++ {
        ev := c.mentat.ConnectivityEvents[i]
        typ := ev["type"]
        ts := ev["timestamp"]
        fmt.Printf("%3d | type: %-25s | timestamp: %s\n", i, typ, ts)
    }

    // Count how many of each type
    counts := make(map[string]int)
    for _, ev := range c.mentat.ConnectivityEvents {
        if t, ok := ev["type"].(string); ok {
            counts[t]++
        }
    }

    fmt.Println("\nEvent type counts:")
    for typ, cnt := range counts {
        fmt.Printf("  %-28s : %d\n", typ, cnt)
    }
}

func (c *MentatCLI) showClumpedErrors(events []new.Event, minutes int) {
    sort.Slice(events, func(i, j int) bool {
        return events[i].Micros < events[j].Micros
    })

    clumpStart := events[0]
    clumpCount := 1
    lastTime := events[0].Micros

    for i := 1; i < len(events); i++ {
        diff := time.Duration(events[i].Micros-lastTime) * time.Microsecond
        if diff <= time.Duration(minutes)*time.Minute {
            clumpCount++
        } else {
            fmt.Printf("[%s] %d errors (router: %s)\n", clumpStart.Timestamp, clumpCount, clumpStart.Router)
            fmt.Printf("   → %s\n\n", shortenLine(clumpStart.Line, 110))
            clumpStart = events[i]
            clumpCount = 1
        }
        lastTime = events[i].Micros
    }

    // Last clump
    fmt.Printf("[%s] %d errors (router: %s)\n", clumpStart.Timestamp, clumpCount, clumpStart.Router)
    fmt.Printf("   → %s\n", shortenLine(clumpStart.Line, 110))
}

func (c *MentatCLI) showEvents(start, end int) {
    fmt.Printf("\nEvents %d to %d:\n", start, end)
    count := 0
    for _, ev := range c.mentat.Events {
        if ev.ID >= start && ev.ID <= end {
            fmt.Printf("%6d | %s | %s | %s\n", ev.ID, ev.Timestamp, ev.Router, shortenLine(ev.Line, 120))
            count++
            if count >= 40 {
                fmt.Println("... (truncated)")
                break
            }
        }
    }
    if count == 0 {
        fmt.Println("No events in that range.")
    }
}

func shortenLine(line string, max int) string {
    if len(line) <= max {
        return line
    }
    return line[:max] + "..."
}


// parseTimeFlexible tries several common formats from the log
func parseTimeFlexible(s string) (time.Time, error) {
    formats := []string{
        "2006-01-02 15:04:05",
        "2006-01-02 15:04",
        "2006-01-02T15:04:05Z",
        time.RFC3339,
    }

    for _, f := range formats {
        if t, err := time.Parse(f, s); err == nil {
            return t, nil
        }
    }
    return time.Time{}, fmt.Errorf("could not parse time")
}
