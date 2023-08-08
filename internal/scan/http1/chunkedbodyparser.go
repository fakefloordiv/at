package http1

import (
	"bytes"
	"errors"
	"fmt"
)

type chunkedBodyScanner struct {
	state       chunkedState
	chunkLength int
}

func newChunkedScanner() *chunkedBodyScanner {
	return new(chunkedBodyScanner)
}

func (c *chunkedBodyScanner) Parse(data []byte) (endsAt int, err error) {
	switch c.state {
	case eChunkLength:
		goto chunkLength
	case eChunkBody:
		goto chunkBody
	case eLastChunk:
		goto lastChunk
	default:
		panic(fmt.Errorf("BUG: unknown state for chunked body: %d", c.state))
	}

chunkLength:
	for i, digit := range data {
		if digit == '\r' {
			continue
		} else if digit == '\n' {
			data = data[i+1:]
			goto chunkLengthEnd
		}

		decoded, ishex := unhex(digit)
		if !ishex {
			return -1, errors.New("bad chunk length char")
		}

		c.chunkLength = (c.chunkLength << 4) | int(decoded)
	}

	return -1, nil

chunkLengthEnd:
	if c.chunkLength > 0 {
		c.state = eChunkBody
		goto chunkBody
	}

	c.state = eLastChunk
	goto lastChunk

chunkBody:
	if len(data) > c.chunkLength {
		rest := data[c.chunkLength:]
		c.chunkLength = 0
		lf := bytes.IndexByte(rest, '\n')
		if lf == -1 {
			return -1, nil
		}

		data = data[lf+1:]
		c.state = eChunkLength
		goto chunkLength
	}

	c.chunkLength -= len(data)

	return -1, nil

lastChunk:
	{
		lf := bytes.IndexByte(data, '\n')
		if lf == -1 {
			return -1, nil
		}

		return lf + 1, nil
	}
}

func unhex(char byte) (byte, bool) {
	switch {
	case '0' <= char && char <= '9':
		return char - '0', true
	case 'a' <= char && char <= 'f':
		return char - 'a' + 10, true
	case 'A' <= char && char <= 'F':
		return char - 'A' + 10, true
	}

	return 0, false
}
