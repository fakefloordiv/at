package connect

import (
	"at/internal/server/tcp"
	"net"
	"time"
)

// TODO network is hardcoded to IPv4, but this should be fixed
const network = "tcp4"

// Connector is a simple, thread-unsafe connection Connector
type Connector struct {
	conns     map[string]tcp.Client
	onConnect func(conn net.Conn) tcp.Client
}

func New(onConnect func(net.Conn) tcp.Client) *Connector {
	return &Connector{
		conns:     map[string]tcp.Client{},
		onConnect: onConnect,
	}
}

func (l *Connector) Set(domain string, conn tcp.Client) {
	l.conns[domain] = conn
}

// Get returns net.Conn corresponding to the domain, or nil if not found
func (l *Connector) Get(domain string) tcp.Client {
	return l.conns[domain]
}

func (l *Connector) Connect(domain string) (tcp.Client, error) {
	conn, err := net.Dial(network, domain)
	if err != nil {
		return nil, err
	}

	client := l.onConnect(conn)
	l.Set(domain, client)

	return client, nil
}

func (l *Connector) Close() {
	for _, conn := range l.conns {
		_ = conn.Close()
	}
}

const (
	defaultReadDeadline = 5 * time.Minute
	defaultReadBuffer   = 4096
)

func OnConnect(conn net.Conn) tcp.Client {
	return tcp.NewClient(conn, defaultReadDeadline, make([]byte, defaultReadBuffer))
}
