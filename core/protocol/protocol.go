package protocol

import (
	"at/internal/server/tcp"
	"encoding/binary"
)

/*
Terms:
- master-server: public server, that accepts connections and forwards data to proxy-server
- master-client: user, that connects to the master-server, and sends data to be forwarded
- proxy-server: server, that receives data from master-server, and forwards it to the proxy-client
- proxy-client: actual local network application
- tunnel: pipe between master-client and proxy-client. This pipe looks in the following way:
  master-client <-> master-server <-> proxy-server <-> proxy-client
- control stream: the first established connection between master- and proxy-server
- data stream: master-server -> proxy-server connection for ordinary data transmission

Protocol scheme:
+--------------+------------------+
| command (u8) | payload (u0-u64) |
+--------------+------------------+
Where:
- command: command enum. See below for the list of all the available
  commands and their descriptions
- payload: variable-length payload, depending on the type

Protocol doesn't provide request-response model, messaging model instead. This means,
that both participants aren't compulsory to respond to each message (but it usually does).
Messages are supposed to be exchanged along exclusive channel, so other data should be
transmitted by own channels.

Valid messages to proxy-server:
- handshake
  - server magic (u64)
- heartbeat
- new stream
- tunnel established
  - addr (u32)
  - port (u16)
- close stream
  - port (u16)

Valid messages to master-server:
- handshake
  - client magic (u64)
- heartbeat
- stream established
  - port (u16)

Assume, that master-server starts listening on the port 100 for proxy-servers. When
proxy-server establishes the connection with master-server by port 100 for the first
time, the connection is considered as control stream. It makes handshake, and starts
waiting for commands. When there is a command to open a new stream, second connection
is established by port 100. Master-server receives a message with a port of a newly
created connection, that is considered to be a new data stream.

Normal communication between master-server and proxy-server looks in the following way:
* proxy-server establishes first connection with a master-server *
- proxy-server -> master-server: Handshake ClientMagic
- master-server -> proxy-server: Handshake ServerMagic
* magic numbers are valid, handshake is completed *
- master-server -> proxy-server: TunnelEstablished addr port
  - where addr, port - a pair of master-server address and port, occupied exclusively by
    proxy-server and used to accept master-clients
* new incoming master-client to the master-server *
- master-server -> proxy-server: NewStream
* proxy-server establishes new connection to both master-server and proxy-client *
- proxy-server -> master-server: StreamEstablished port
* data starts transferring through the tunnel *
if master-client disconnects:
  - master-server -> proxy-server: CloseStream port
if proxy-server data stream actively closes:
  - drop connection also with a master-client
*/

const (
	Handshake byte = iota + 1
	Heartbeat
	NewStream
	StreamEstablished
	CloseStream
	TunnelEstablished
)

const (
	ClientMagic = uint64(9246843611175041024)
	ServerMagic = uint64(7936407530654337405)
)

type Message struct {
	Command byte
	Addr    uint32
	Port    uint16
	Magic   uint64
}

func (m *Message) Send(client tcp.Client, buff []byte) error {
	switch m.Command {
	case Handshake:
		return client.Write(append(buff, Handshake))
	case Heartbeat:
		return client.Write(append(buff, Heartbeat))
	case NewStream:
		return client.Write(append(buff, NewStream))
	case CloseStream:
		buff = append(buff, CloseStream)
		buff = binary.LittleEndian.AppendUint16(buff, m.Port)

		return client.Write(buff)
	case StreamEstablished:
		buff = append(buff, TunnelEstablished)
		buff = binary.LittleEndian.AppendUint16(buff, m.Port)

		return client.Write(buff)
	case TunnelEstablished:
		buff = append(buff, TunnelEstablished)
		buff = binary.LittleEndian.AppendUint32(buff, m.Addr)
		buff = binary.LittleEndian.AppendUint16(buff, m.Port)

		return client.Write(buff)
	default:
		panic("BUG: send(): unknown command")
	}
}

type Parser struct {
	client tcp.Client
	buffer []byte
}

func NewParser(client tcp.Client) *Parser {
	return &Parser{
		client: client,
		buffer: make([]byte, 0, 64),
	}
}

func (p *Parser) Read() (msg Message, err error) {
	data, err := p.client.Read()
	if err != nil || len(data) == 0 {
		return msg, err
	}

	msg.Command = data[0]
	p.client.Unread(data[1:])

	switch msg.Command {
	case Handshake:
		magic, err := p.readN(8)
		if err != nil {
			return msg, err
		}

		msg.Magic = binary.LittleEndian.Uint64(magic)

		return msg, nil
	case Heartbeat:
		return msg, nil
	case NewStream:
		return msg, nil
	case StreamEstablished:
		port, err := p.readN(2)
		if err != nil {
			return msg, err
		}

		msg.Port = binary.LittleEndian.Uint16(port)

		return msg, nil
	case CloseStream:
		port, err := p.readN(2)
		if err != nil {
			return msg, err
		}

		msg.Port = binary.LittleEndian.Uint16(port)

		return msg, nil
	case TunnelEstablished:
		addr, err := p.readN(4)
		if err != nil {
			return msg, err
		}

		msg.Addr = binary.LittleEndian.Uint32(addr)

		port, err := p.readN(2)
		if err != nil {
			return msg, err
		}

		msg.Port = binary.LittleEndian.Uint16(port)

		return msg, nil
	default:
		return msg, ErrUnknownCommand
	}
}

func (p *Parser) readN(n int) ([]byte, error) {
	for {
		data, err := p.client.Read()
		if err != nil {
			return nil, err
		}

		if len(p.buffer)+len(data) >= n {
			value := append(p.buffer, data[:n-len(p.buffer)]...)
			p.client.Unread(data[n-len(p.buffer):])
			p.buffer = p.buffer[:0]

			return value, nil
		}

		p.buffer = append(p.buffer, data...)
	}
}
