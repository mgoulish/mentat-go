package connectivity

import (
    "fmt"
    "regexp"
    "sort"
    "time"

    "github.com/mgoulish/mentat/internal/debug"
    "github.com/mgoulish/mentat/internal/new"
    "github.com/mgoulish/mentat/internal/utils"
)

var (
    
    tsRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+)`)

    // Router startup
    routerStartedRegex      = regexp.MustCompile(`Router started in (Interior|Edge) mode`)
    routerVersionRegex      = regexp.MustCompile(`Version: ([\d\.\-rh]+)`)
    routerEngineRegex       = regexp.MustCompile(`Router Engine Instantiated`)

    // SSL
    sslProfileRegex         = regexp.MustCompile(`Created SSL Profile with name (\S+)`)

    // Configured Listener / Connector
    configuredListenerRegex = regexp.MustCompile(`Configured\s+Listener:\s*(:?\S+).*?role=([^,\s]+)`)
    configuredConnectorRegex = regexp.MustCompile(`Configured\s+Connector:\s*([^:\s]+):(\d+).*?role=([^,\s]+)`)

    // HTTP
    httpListeningRegex      = regexp.MustCompile(`Listening for HTTP on :(\d+)`)

    // SERVER
    serverListeningRegex    = regexp.MustCompile(`Listening on (.*?:\d+)`)
    acceptedConnectionRegex = regexp.MustCompile(`\[C\d+\] Accepted connection to (.*?) from (.*?)`)

    // Legacy TCP
    tcpListenerRegex        = regexp.MustCompile(`Configured TcpListener .*?for ([^,]+):(\d+)`)
    tcpConnectorRegex       = regexp.MustCompile(`Configured TcpConnector .*?for ([^,]+):(\d+)`)

    // Client listener
    clientListenerRegex     = regexp.MustCompile(`Listener\s+([^:]+):(\d+):\s*listening for client connections`)
    clientListenerStoppedRegex = regexp.MustCompile(`Listener\s+([^:]+):(\d+):\s*stopped listening for client connections`)

    // FLOW_LOG
    flowLogConnectorRegex   = regexp.MustCompile(`CONNECTOR \[.*?\] BEGIN .*?destHost=([^ ]+) .*?destPort=(\d+)`)
    flowLogListenerRegex    = regexp.MustCompile(`LISTENER \[.*?\] BEGIN .*?destHost=([^ ]+) .*?destPort=(\d+)`)

    // ROUTER_LS
    routerLSNextHopsRegex   = regexp.MustCompile(`Computed next hops:`)
    routerLSCostsRegex      = regexp.MustCompile(`Computed costs:`)
    routerLSLinkLostRegex   = regexp.MustCompile(`Link to Neighbor Router Lost`)

    // Errors
    errorRegex              = regexp.MustCompile(`(?i)(error|ERROR|failed|fail|framing-error|Unknown protocol)`)
    
    //interRouterLinkRegex = regexp.MustCompile(`prd-wyn-skupper-router-\S+`)
    interRouterLinkRegex = regexp.MustCompile(`prd-wyn-skupper-router-\S+`)
    mongoServiceRegex    = regexp.MustCompile(`prd-mc-rs-mdb-\S+`)
)

func ReadConnectivityEvents(mentat *new.Mentat) {
    debug.Info("Extracting connectivity events from logs...")

    count := 0
    for _, site := range mentat.Sites {
        for _, router := range site.Routers {
            allEvents := append(router.CurrentEvents, router.PreviousEvents...)

            for _, ev := range allEvents {
                if data := parseLogLine(ev.Line, site.Name, router.Name); data != nil {
                    mentat.ConnectivityEvents = append(mentat.ConnectivityEvents, data)
                    count++
                }
            }
        }
    }

    debug.Info(fmt.Sprintf("Extracted %d connectivity events", count))
}

func parseLogLine(line, site, router string) new.ConnectivityEvent {
    tsMatch := tsRegex.FindStringSubmatch(line)
    if len(tsMatch) == 0 {
        return nil
    }
    timestamp := tsMatch[1]
    micros, _ := utils.StringToMicrosecondsSinceEpoch(timestamp + " +0000")

    // Router startup
    if m := routerStartedRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "router_started", "timestamp": timestamp, "microseconds": micros, "mode": m[1], "site": site, "router": router}
    }
    if routerVersionRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "router_version", "timestamp": timestamp, "microseconds": micros, "site": site, "router": router}
    }
    if routerEngineRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "router_engine_instantiated", "timestamp": timestamp, "microseconds": micros, "site": site, "router": router}
    }

    // SSL Profile
    if m := sslProfileRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "ssl_profile_created", "timestamp": timestamp, "microseconds": micros, "name": m[1], "site": site, "router": router}
    }

    // Configured Listener
    if m := configuredListenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "configured_listener", "timestamp": timestamp, "microseconds": micros, "listener_name": m[1], "role": m[2], "site": site, "router": router}
    }

    // Configured Connector
    if m := configuredConnectorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "configured_connector", "timestamp": timestamp, "microseconds": micros, "connector_name": m[1], "port": m[2], "role": m[3], "site": site, "router": router}
    }

    // HTTP Listener
    if m := httpListeningRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "http_listener", "timestamp": timestamp, "microseconds": micros, "port": m[1], "site": site, "router": router}
    }

    // Server Listening
    if m := serverListeningRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "server_listening", "timestamp": timestamp, "microseconds": micros, "address": m[1], "site": site, "router": router}
    }

    // Accepted Connection
    if m := acceptedConnectionRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "accepted_connection", "timestamp": timestamp, "microseconds": micros, "to": m[1], "from": m[2], "site": site, "router": router}
    }

    // Legacy TCP
    if m := tcpListenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "tcp_listener", "timestamp": timestamp, "microseconds": micros, "service": m[1], "port": m[2], "site": site, "router": router}
    }
    if m := tcpConnectorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "tcp_connector", "timestamp": timestamp, "microseconds": micros, "service": m[1], "port": m[2], "site": site, "router": router}
    }

    // Client listener START
    if m := clientListenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{
            "type":        "client_listener",
            "timestamp":   timestamp,
            "microseconds": micros,
            "listener_name": m[1],
            "port":        m[2],
            "site":        site,
            "router":      router,
        }
    }

    // Client listener STOPPED
    if m := clientListenerStoppedRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{
            "type":        "client_listener_stopped",
            "timestamp":   timestamp,
            "microseconds": micros,
            "listener_name": m[1],
            "port":        m[2],
            "site":        site,
            "router":      router,
        }
    }

    // FLOW_LOG
    if m := flowLogConnectorRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "flow_connector", "timestamp": timestamp, "microseconds": micros, "dest_host": m[1], "dest_port": m[2], "site": site, "router": router}
    }
    if m := flowLogListenerRegex.FindStringSubmatch(line); m != nil {
        return new.ConnectivityEvent{"type": "flow_listener", "timestamp": timestamp, "microseconds": micros, "dest_host": m[1], "dest_port": m[2], "site": site, "router": router}
    }

    // ROUTER_LS
    if routerLSNextHopsRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "router_ls_next_hops", "timestamp": timestamp, "microseconds": micros, "site": site, "router": router}
    }
    if routerLSCostsRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "router_ls_costs", "timestamp": timestamp, "microseconds": micros, "site": site, "router": router}
    }
    if routerLSLinkLostRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "router_link_lost", "timestamp": timestamp, "microseconds": micros, "site": site, "router": router}
    }

    // Errors
    if errorRegex.MatchString(line) {
        return new.ConnectivityEvent{"type": "error", "timestamp": timestamp, "microseconds": micros, "message": line, "site": site, "router": router}
    }

    return nil
}

// ================================================================
// Connectivity State Snapshot
// ================================================================

// ConnectivityState holds the router's connectivity picture at any chosen moment.
type ConnectivityState struct {
	Timestamp        time.Time
	ConnectedRouters []string // inter-router neighbors
	ActiveServices   []string // MongoDB services via flow/tcp listeners
	ActiveClients    int      // accepted connections
}

// stateTracker tracks live state while replaying events.
type stateTracker struct {
	connectedRouters map[string]bool
	activeServices   map[string]bool
	activeClients    int
}


// StateAt returns the connectivity state at or immediately before the given timestamp.
func StateAt(events []new.ConnectivityEvent, t time.Time) *ConnectivityState {
	tr := &stateTracker{
		connectedRouters: make(map[string]bool),
		activeServices:   make(map[string]bool),
	}

	target := t.UTC()

	for _, ev := range events {
		micros, ok := ev["microseconds"].(int64)
		if !ok {
			continue
		}
		eventTime := time.UnixMicro(micros).UTC()

		if eventTime.After(target) && eventTime.Sub(target) > time.Second {
			break
		}

		typ, _ := ev["type"].(string)

		switch typ {
		case "flow_connector", "configured_connector":
			// Look for inter-router link (wyn router) in any string field
			for _, v := range ev {
				if s, ok := v.(string); ok {
					if h := interRouterLinkRegex.FindString(s); h != "" {
						tr.connectedRouters[h] = true
						break
					}
				}
			}

		case "flow_listener", "tcp_listener", "client_listener", "configured_listener":
			if svc := extractService(ev); svc != "" {
				tr.activeServices[svc] = true
			}

		case "accepted_connection":
			tr.activeClients++
		}
	}

	routers := make([]string, 0, len(tr.connectedRouters))
	for r := range tr.connectedRouters {
		routers = append(routers, r)
	}
	sort.Strings(routers)

	services := make([]string, 0, len(tr.activeServices))
	for s := range tr.activeServices {
		services = append(services, s)
	}
	sort.Strings(services)

	return &ConnectivityState{
		Timestamp:        t,
		ConnectedRouters: routers,
		ActiveServices:   services,
		ActiveClients:    tr.activeClients,
	}
}



func extractService(ev new.ConnectivityEvent) string {
	for _, key := range []string{"service", "listener_name", "dest_host"} {
		if v, ok := ev[key]; ok && v != nil {
			s := fmt.Sprintf("%v", v)
			if m := mongoServiceRegex.FindString(s); m != "" {
				return m
			}
		}
	}
	return ""
}

// Pretty-print for easy reading in CLI or TUI
func (s *ConnectivityState) String() string {
	return fmt.Sprintf(`=== Connectivity State at %s ===
Connected routers : %v
Active services   : %v
Active clients    : %d
`,
		s.Timestamp.Format(time.RFC3339),
		s.ConnectedRouters,
		s.ActiveServices,
		s.ActiveClients)
}
