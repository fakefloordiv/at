package master

import (
	"at/internal/address"
	"at/internal/server/tcp"
	"context"
	"encoding/binary"
	"log"
	"net"
	"sync"
)

type ProxyListener struct {
	// conns is considered to be a one-dimensional hash-map by storing already a pair of
	// IP and port, so we don't allocate many more nested second-dimensional maps. Control
	// streams are always stored with port=0, so we can look up whether the connection
	conns map[address.Addr]net.Conn
	mu    sync.RWMutex
}

func NewProxyListener() *ProxyListener {
	return &ProxyListener{
		conns: make(map[address.Addr]net.Conn),
	}
}

func (p *ProxyListener) Get(addr address.Addr) (net.Conn, bool) {
	p.mu.RLock()
	conn, found := p.conns[addr]
	p.mu.RUnlock()

	return conn, found
}

func (p *ProxyListener) Listen(sock net.Listener) {
	// TODO: this method is supposed to be running as a stand-alone goroutine. So in case
	//  our proxy listener socket is down, how can we signalize about it? Or maybe just log
	//  it, and silently try to bind it again? In case we do nothing, it can be a bit embarrassing.
	for {
		err := tcp.Run(context.Background(), sock, func(conn net.Conn) {
			remote := conn.RemoteAddr().(*net.TCPAddr).AddrPort()
			// TODO: check whether connection is DEFINITELY ip4
			remoteIP := remote.Addr().As4()
			addr := address.Addr{
				Ip: binary.LittleEndian.Uint32(remoteIP[:]),
			}

			if p.lookupAddr(addr) {
				addr.Port = remote.Port()
			} else {
				// in case of control stream, just keep the port zeroed
				p.handleControlStream(conn)
			}

			p.addConn(conn, addr)
		})

		log.Println("error: tcp: proxy listener:", err)
		log.Println("tcp: proxy listener: re-starting listener")
		// TODO: in case proxy listener socket is somehow dead and unusable anymore, we'll just
		//  spam in the console with these two lines. Fix this somehow
		// TODO: we also need to tell all others that proxy server listener is dead in case error
		//  is unrecoverable by re-binding the socket after errors. The best choice is to write
		//  RIP to logs and loudly lay down, as it's better, than implicitly not accepting data
		//  streams
	}
}

func (p *ProxyListener) handleControlStream(conn net.Conn) {
	// here we must initialize a brand new tunnel
}

func (p *ProxyListener) addConn(conn net.Conn, addr address.Addr) {
	p.mu.Lock()
	p.conns[addr] = conn
	p.mu.Unlock()
}

// lookupAddr looks up in the conns, whether an addr is presented
func (p *ProxyListener) lookupAddr(addr address.Addr) bool {
	p.mu.RLock()
	_, found := p.conns[addr]
	p.mu.RUnlock()

	return found
}
