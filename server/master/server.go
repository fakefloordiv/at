package master

import "net"

type Server struct {
	masterClients []net.Conn
}
