package connectivity

import (
    "fmt"
    "regexp"

    "github.com/mgoulish/mentat/internal/debug"
    "github.com/mgoulish/mentat/internal/new"
    "github.com/mgoulish/mentat/internal/utils"
)

var (
    tsRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+)`)

    connectorRegex    = regexp.MustCompile(`Configured\s+Connector:\s*([^:\s]+):(\d+).*?role=([^,\s]+)`)
    listenerRegex     = regexp.MustCompile(`Configured\s+Listener:\s*:?(\d+).*?role=([^,\s]+)`)
    httpRegex         = regexp.MustCompile(`Listening for HTTP on :(\d+)`)
    clientRegex       = regexp.MustCompile(`Listener\s+([^:]+):(\d+).*?listening for client connections on .*?:(\d+)`)
    tcpListenerRegex  = regexp.MustCompile(`Configured TcpListener .*?for ([^:]+):(\d+)`)
    tcpConnectorRegex = regexp.MustCompile(`Configured TcpConnector .*?for ([^:]+):(\d+)`)
    serverListenRegex = regexp.MustCompile(`Listening on (.*?:\d+)`)
    errorRegex        = regexp.MustCompile(`(?i)(error|ERROR|failed|fail|exception)`)
)

func ReadConnectivityEvents(mentat *new.Mentat) {
    debug.Info("Extracting connectivity events from logs...")

    count := 0
    httpCount := 0
    serverCount := 0

    for _, site := range mentat.Sites {
        for _, router := range site.Routers {
            allEvents := append(router.CurrentEvents, router.PreviousEvents...)

            for _, ev := range allEvents {
                if data := parseLogLine(ev.Line, site.Name, router.Name); data != nil {
                    mentat.ConnectivityEvents = append(mentat.ConnectivityEvents, data)
                    count++

                    if t, ok := data["type"].(string); ok {
                        if t == "http_listener" {
                            httpCount++
                        }
                        if t == "server_listening" {
                            serverCount++
                        }
                    }
                }
            }
        }
    }

    debug.Info(fmt.Sprintf("Extracted %d connectivity events", count))
    debug.Info(fmt.Sprintf("HTTP listeners found: %d", httpCount))
    debug.Info(fmt.Sprintf("Server listening entries found: %d", serverCount))
}

func parseLogLine(line, site, router string) new.ConnectivityEvent {
    tsMatch := tsRegex.FindStringSubmatch(line)
    if len(tsMatch) == 0 {
        return nil
    }
    timestamp := tsMatch[1]
    micros, _ := utils.StringToMicrosecondsSinceEpoch(timestamp + " +0000")

    if m := connectorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "configured_connector", "timestamp": timestamp, "microseconds": micros, "connector_name": m[1], "port": m[2], "role": m[3], "site": site, "router": router}
    }
    if m := listenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "configured_listener", "timestamp": timestamp, "microseconds": micros, "listener_name": ":" + m[1], "port": m[1], "role": m[2], "site": site, "router": router}
    }
    if m := httpRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "http_listener", "timestamp": timestamp, "microseconds": micros, "port": m[1], "site": site, "router": router}
    }
    if m := clientRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "listening_for_client", "timestamp": timestamp, "microseconds": micros, "listener_name": m[1], "port": m[2], "target_port": m[3], "site": site, "router": router}
    }
    if m := tcpListenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "tcp_listener", "timestamp": timestamp, "microseconds": micros, "service": m[1], "port": m[2], "site": site, "router": router}
    }
    if m := tcpConnectorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "tcp_connector", "timestamp": timestamp, "microseconds": micros, "service": m[1], "port": m[2], "site": site, "router": router}
    }
    if m := serverListenRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "server_listening", "timestamp": timestamp, "microseconds": micros, "address": m[1], "site": site, "router": router}
    }
    if m := errorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "error", "timestamp": timestamp, "microseconds": micros, "message": line, "site": site, "router": router}
    }

    return nil
}
