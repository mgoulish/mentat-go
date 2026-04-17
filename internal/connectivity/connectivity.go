package connectivity

import (
    "fmt"
    "regexp"
    "sort"
    "strings"
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

    // Start with a base event that always has the critical fields
    event := new.ConnectivityEvent{
        "type":         "unknown",
        "timestamp":    timestamp,
        "microseconds": micros,
        "site":         site,
        "router":       router,
    }

    // Router startup
    if m := routerStartedRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "router_started"
        event["mode"] = m[1]
        return event
    }
    if routerVersionRegex.MatchString(line) {
        event["type"] = "router_version"
        return event
    }
    if routerEngineRegex.MatchString(line) {
        event["type"] = "router_engine_instantiated"
        return event
    }

    // SSL Profile
    if m := sslProfileRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "ssl_profile_created"
        event["name"] = m[1]
        return event
    }

    // Configured Listener
    if m := configuredListenerRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "configured_listener"
        event["listener_name"] = m[1]
        event["role"] = m[2]
        return event
    }

    // Configured Connector
    if m := configuredConnectorRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "configured_connector"
        event["connector_name"] = m[1]
        event["port"] = m[2]
        event["role"] = m[3]
        return event
    }

    // HTTP Listener
    if m := httpListeningRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "http_listener"
        event["port"] = m[1]
        return event
    }

    // Server Listening
    if m := serverListeningRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "server_listening"
        event["address"] = m[1]
        return event
    }

    // Accepted Connection
    if m := acceptedConnectionRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "accepted_connection"
        event["to"] = m[1]
        event["from"] = m[2]
        return event
    }

    // Legacy TCP
    if m := tcpListenerRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "tcp_listener"
        event["service"] = m[1]
        event["port"] = m[2]
        return event
    }
    if m := tcpConnectorRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "tcp_connector"
        event["service"] = m[1]
        event["port"] = m[2]
        return event
    }

    // Client listener START
    if m := clientListenerRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "client_listener"
        event["listener_name"] = m[1]
        event["port"] = m[2]
        return event
    }

    // Client listener STOPPED
    if m := clientListenerStoppedRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "client_listener_stopped"
        event["listener_name"] = m[1]
        event["port"] = m[2]
        return event
    }

    // FLOW_LOG
    if m := flowLogConnectorRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "flow_connector"
        event["dest_host"] = m[1]
        event["dest_port"] = m[2]
        return event
    }
    if m := flowLogListenerRegex.FindStringSubmatch(line); m != nil {
        event["type"] = "flow_listener"
        event["dest_host"] = m[1]
        event["dest_port"] = m[2]
        return event
    }

    // ROUTER_LS
    if routerLSNextHopsRegex.MatchString(line) {
        event["type"] = "router_ls_next_hops"
        return event
    }
    if routerLSCostsRegex.MatchString(line) {
        event["type"] = "router_ls_costs"
        return event
    }
    if routerLSLinkLostRegex.MatchString(line) {
        event["type"] = "router_link_lost"
        return event
    }

    // Errors
    if errorRegex.MatchString(line) {
        event["type"] = "error"
        event["message"] = line
        return event
    }

    // If we reach here, return the base event with "unknown" type
    return event
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
		// Extract microseconds reliably
		var micros int64 = 0
		switch v := ev["microseconds"].(type) {
		case int64:
			micros = v
		case float64:
			micros = int64(v)
		case int:
			micros = int64(v)
		}

		if micros == 0 {
			continue
		}

		eventTime := time.UnixMicro(micros).UTC()
		if eventTime.After(target) && eventTime.Sub(target) > 10*time.Minute {
			break
		}

		typ, _ := ev["type"].(string)
		if typ == "" {
			continue
		}

		switch {
		case strings.Contains(typ, "link_lost") || strings.Contains(typ, "Link Lost"):
			// Clear connected routers on explicit link loss
			tr.connectedRouters = make(map[string]bool)

		case strings.Contains(typ, "connector") || 
		     strings.Contains(typ, "next_hops") || 
		     strings.Contains(typ, "router_started"):
			// Re-add any router name we see
			for _, v := range ev {
				if s, ok := v.(string); ok {
					if strings.Contains(s, "dor") || strings.Contains(s, "wyn") || strings.Contains(s, "skupper-router") {
						tr.connectedRouters[s] = true
					}
				}
			}

		case strings.Contains(typ, "listener") || strings.Contains(typ, "service") || strings.Contains(typ, "flow_"):
			if svc := extractService(ev); svc != "" {
				tr.activeServices[svc] = true
			}

		case strings.Contains(typ, "accepted_connection") || strings.Contains(typ, "server_listening"):
			tr.activeClients++
		}
	}

	// Build output lists
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


func (s *ConnectivityState) String() string {
	return fmt.Sprintf(`Connected routers : %v
Active services   : %v
Active clients    : %d
`, s.ConnectedRouters, s.ActiveServices, s.ActiveClients)
}


