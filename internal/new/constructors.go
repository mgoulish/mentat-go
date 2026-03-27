package new

func NewMentat(root string) *Mentat {
    return &Mentat{
        Root:               root,
        Sites:              []Site{},
        Events:             []Event{},
        ConnectivityEvents: []ConnectivityEvent{},
    }
}

func NewSite(name, root string) Site {
    return Site{
        Name:      name,
        Root:      root,
        Listeners: []Listener{},
        Connectors: []Connector{},
        Routers:   []Router{},
    }
}

func NewRouter(name, site, nickname string) Router {
    return Router{
        Name:           name,
        Nickname:       nickname,
        Site:           site,
        CurrentEvents:  []Event{},
        PreviousEvents: []Event{},
    }
}

func NewListener() Listener {
    return Listener{Role: "normal"}
}

func NewConnector() Connector {
    return Connector{Role: "normal"}
}

func NewEvent(eventType, timestamp string) Event {
    return Event{
        Type:      eventType,
        Timestamp: timestamp,
    }
}
