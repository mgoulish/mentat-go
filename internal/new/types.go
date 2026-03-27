package new

type Mentat struct {
    Root               string
    Sites              []Site
    Events             []Event
    ConnectivityEvents []ConnectivityEvent
}

type Site struct {
    Name       string
    Root       string
    IngressHost string
    Routers    []Router
    Listeners  []Listener
    Connectors []Connector
}

type Router struct {
    Name           string
    Nickname       string
    Site           string
    CurrentEvents  []Event
    PreviousEvents []Event
}

type Listener struct {
    Name string
    Port int
    Role string
}

type Connector struct {
    Host string
    Name string
    Port int
    Role string
}

type Event struct {
    Type       string
    Timestamp  string
    Micros     int64
    ID         int
    Line       string
    FilePath   string
    LineNumber int
    Router     string
    Site       string
}

type ConnectivityEvent map[string]any
