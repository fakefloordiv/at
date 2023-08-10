package tcp

import (
	"context"
	"log"
	"net"
	"sync"
)

func Run(ctx context.Context, sock net.Listener, onConn func(conn net.Conn)) error {
	wg := new(sync.WaitGroup)

	for {
		if err := ctx.Err(); err != nil {
			wg.Wait()

			return err
		}

		conn, err := sock.Accept()
		if err != nil {
			log.Println("error accepting a connection:", err)
			continue
		}

		wg.Add(1)
		go func() {
			onConn(conn)
			wg.Done()
		}()
	}
}
