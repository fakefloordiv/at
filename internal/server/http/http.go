package http

import (
	"at/internal/connect"
	"at/internal/scan"
	"at/internal/server/tcp"
	"github.com/indigo-web/utils/arena"
	"strings"
)

type Server struct {
	client    tcp.Client
	scanner   scan.Scanner
	connector *connect.Connector
	buffer    *arena.Arena[byte]
}

func New(
	client tcp.Client, scanner scan.Scanner, connector *connect.Connector, buffer *arena.Arena[byte],
) *Server {
	return &Server{
		client:    client,
		scanner:   scanner,
		connector: connector,
		buffer:    buffer,
	}
}

func (s *Server) Serve() {
	defer func() {
		_ = s.client.Close()
		s.connector.Close()
	}()

	var forwardTo string

amass:
	for {
		data, err := s.client.Read()
		if err != nil {
			return
		}

		host, endsAt, err := s.scanner.Scan(data)
		if err != nil {
			return
		}

		// basically, there are three options now:

		// 1) we received the whole request all at once
		if endsAt != -1 {
			if !s.buffer.Append(data[:endsAt]...) {
				return
			}

			s.client.Unread(data[endsAt:])
			if err = s.send(host, s.buffer.Finish()); err != nil {
				return
			}

			s.buffer.Clear()
			continue
		}

		// 2) we finally received Host header value, but not the whole request yet
		if len(host) > 0 {
			forwardTo = host
			goto transit
		}

		// 3) no whole request, no Host, no fun. Just save it and keep going
		if !s.buffer.Append(data...) {
			// in case client exceeds forwarder's buffer size, just drop the connection.
			// There's nothing else we can do in this situation
			return
		}
	}

transit:
	for {
		data, err := s.client.Read()
		if err != nil {
			return
		}

		_, endsAt, err := s.scanner.Scan(data)
		if err != nil {
			return
		}

		if endsAt != -1 {
			if !s.drain(forwardTo, data, endsAt) {
				return
			}

			goto amass
		}

		err = s.send(forwardTo, data)
		if err != nil {
			return
		}
	}
}

func (s *Server) drain(to string, data []byte, endsAt int) (ok bool) {
	piece, leftData := data[:endsAt], data[endsAt:]
	s.client.Unread(leftData)
	err := s.send(to, piece)

	return err == nil
}

func (s *Server) send(to string, data []byte) (err error) {
	to = stripWWW(to)
	host := s.connector.Get(to)
	if host == nil {
		host, err = s.connector.Connect(to)
		if err != nil {
			return err
		}

		go func() {
			for {
				data, err := host.Read()
				if err != nil {
					return
				}

				if err := s.client.Write(data); err != nil {
					return
				}
			}
		}()
	}

	return host.Write(data)
}

func stripWWW(domain string) string {
	return strings.TrimPrefix(domain, "www.")
}
