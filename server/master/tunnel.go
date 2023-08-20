package master

import "net"

// Tunnel binds master-client and proxy-server together
type Tunnel struct {
	proxyServer net.Conn
}

func NewTunnel(proxy net.Conn) Tunnel {
	return Tunnel{
		proxyServer: proxy,
	}
}

func (t Tunnel) Bind() net.Addr {

}

func (t Tunnel) Serve() {
	
}
