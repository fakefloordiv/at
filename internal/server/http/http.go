package http

import (
	"at/internal/connect"
	"at/internal/scan"
	"at/internal/server/tcp"
	"fmt"
	"github.com/indigo-web/utils/arena"
	"strings"
)

type Server struct {
	client        tcp.Client
	scanner       scan.Scanner
	connector     *connect.Connector
	forwardTo     string
	headersBuffer *arena.Arena[byte]
}

func New(client tcp.Client, connector *connect.Connector, headersBuff *arena.Arena[byte]) *Server {
	return &Server{
		client:        client,
		connector:     connector,
		headersBuffer: headersBuff,
	}
}

func (s *Server) Serve() {
	defer func() {
		_ = s.client.Close()
		s.connector.Close()
	}()

	state := eHeaders

	for {
		data, err := s.client.Read()
		if err != nil {
			return
		}

		switch state {
		case eHeaders:
			report, done, rest, err := s.scanner.Scan(data)
			if err != nil {
				return
			}

			if !s.headersBuffer.Append(data...) {
				// in case client exceeds forwarder's buffer size, just drop the connection
				return
			}

			s.client.Unread(rest)

			if done {
				s.forwardTo = stripWWW(report.Host)
				if err = s.send(s.headersBuffer.Finish()); err != nil {
					return
				}

				if report.ContentLength > 0 || report.IsChunked {
					state = eBody
				}
			}
		case eBody:
			endsAt, err := s.scanner.Body(data)
			if err != nil {
				return
			}

			if err = s.send(data); err != nil {
				return
			}

			if endsAt == -1 {
				s.client.Unread(data[endsAt:])
				state = eHeaders
			}
		default:
			panic(fmt.Errorf("BUG: unknown HTTP server state: %d", state))
		}
	}
}

func (s *Server) send(data []byte) (err error) {
	client := s.connector.Get(s.forwardTo)
	if client == nil {
		client, err = s.connector.Connect(s.forwardTo)
		if err != nil {
			return err
		}
	}
}

func stripWWW(domain string) string {
	return strings.TrimPrefix(domain, "www.")
}
