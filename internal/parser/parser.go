package parser

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sort"

    "github.com/mgoulish/mentat/internal/debug"
    "github.com/mgoulish/mentat/internal/new"
    "github.com/mgoulish/mentat/internal/utils"
)

var timestampRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{6})`)

func ReadEvents(mentat *new.Mentat) {
    debug.Info("Starting to read router log files...")

    eventID := 1

    for i := range mentat.Sites {
        site := &mentat.Sites[i]
        for j := range site.Routers {
            router := &site.Routers[j]

            routerDir := filepath.Join(site.Root, "pods", router.Name, "logs")
            currentLog := filepath.Join(routerDir, "router-logs.txt")
            previousLog := filepath.Join(routerDir, "router-logs-previous.txt")

            // Read current log
            events := readRouterLog(currentLog, router.Name, site.Name)
            for k := range events {
                events[k].ID = eventID
                eventID++
                router.CurrentEvents = append(router.CurrentEvents, events[k])
                mentat.Events = append(mentat.Events, events[k])
            }

            // Read previous log if it exists
            if _, err := os.Stat(previousLog); err == nil {
                events := readRouterLog(previousLog, router.Name, site.Name)
                for k := range events {
                    events[k].ID = eventID
                    eventID++
                    router.PreviousEvents = append(router.PreviousEvents, events[k])
                    mentat.Events = append(mentat.Events, events[k])
                }
            }
        }
    }

    // Sort all events chronologically
    sort.Slice(mentat.Events, func(i, j int) bool {
        return mentat.Events[i].Micros < mentat.Events[j].Micros
    })

    debug.Info(fmt.Sprintf("Read %d total events from all router logs", len(mentat.Events)))
}

func readRouterLog(logPath, routerName, siteName string) []new.Event {
    var events []new.Event

    file, err := os.Open(logPath)
    if err != nil {
        debug.Info(fmt.Sprintf("Could not open log file %s: %v", logPath, err))
        return events
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        line := scanner.Text()
        lineNum++

        if match := timestampRegex.FindStringSubmatch(line); match != nil {
            timestamp := match[1]
            micros, err := utils.StringToMicrosecondsSinceEpoch(timestamp)
            if err != nil {
                micros = 0
            }

            event := new.NewEvent("log_line", timestamp)
            event.Micros = micros
            event.Line = line
            event.FilePath = logPath
            event.LineNumber = lineNum
            event.Router = routerName
            event.Site = siteName

            events = append(events, event)
        }
    }

    return events
}
