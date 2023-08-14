package main

import (
	"at/internal/connect"
	"at/internal/scan/http1"
	"at/internal/server/http"
	"at/internal/server/tcp"
	"context"
	"fmt"
	"github.com/indigo-web/utils/arena"
	"net"
	"time"
)

const (
	network       = "tcp4"
	addr          = "0.0.0.0:8000"
	readDeadline  = 3 * time.Minute
	writeDeadline = 1 * time.Minute
)

func main() {
	sock, err := net.Listen(network, addr)
	if err != nil {
		fmt.Println("error: listen:", err)
		return
	}

	fmt.Println("Starting on", network, addr)

	err = tcp.Run(context.Background(), sock, func(conn net.Conn) {
		client := tcp.NewClient(conn, readDeadline, writeDeadline, make([]byte, 4096))
		scanner := http1.NewScanner()
		connector := connect.New(func(conn net.Conn) tcp.Client {
			return tcp.NewClient(conn, readDeadline, writeDeadline, make([]byte, 4096))
		})
		buffer := arena.NewArena[byte](4*1024 /* 4kb */, 64*1024 /* 64kb */)
		server := http.New(client, scanner, connector, buffer)
		server.Serve()
	})
	if err != nil {
		fmt.Println("error: tcp:", err)
	}
}
