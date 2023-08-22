package master

import (
	"at/internal/server/tcp"
	"context"
	"net"
)

type Server struct {
	masterClients []net.Conn
	proxyServers map[]
}

func (s *Server) Serve(masterSock, proxySock net.Listener) error {
	return tcp.Run(context.Background(), masterSock, func(conn net.Conn) {
		s.masterClients = append(s.masterClients, conn)
	})
}

func (s *Server) serveProxyServers(proxySock net.Listener) {
	for {
		_ = tcp.Run(context.Background(), proxySock, func(conn net.Conn) {
			s.proxyServers = append(s.proxyServers, conn)
		})
	}
}

func
