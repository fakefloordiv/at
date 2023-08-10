package tcp

import (
	"net"
	"time"
)

type Client interface {
	Write([]byte) error
	Read() ([]byte, error)
	Unread([]byte)
	Close() error
}

type client struct {
	conn                        net.Conn
	readDeadline, writeDeadline time.Duration
	buff                        []byte
	unread                      []byte
}

func NewClient(conn net.Conn, rDeadline, wDeadline time.Duration, buff []byte) Client {
	return &client{
		conn:          conn,
		readDeadline:  rDeadline,
		writeDeadline: wDeadline,
		buff:          buff,
	}
}

func (c *client) Write(data []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline)); err != nil {
		return err
	}

	_, err := c.conn.Write(data)

	return err
}

func (c *client) Read() ([]byte, error) {
	if len(c.unread) > 0 {
		data := c.unread
		c.unread = nil

		return data, nil
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(c.readDeadline)); err != nil {
		return nil, err
	}

	n, err := c.conn.Read(c.buff)

	return c.buff[:n], err
}

func (c *client) Unread(data []byte) {
	c.unread = data
}

func (c *client) Close() error {
	return c.conn.Close()
}
