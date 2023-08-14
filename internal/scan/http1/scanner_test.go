package http1

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScanner(t *testing.T) {
	t.Run("simple get", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\n\r\n"
		scan := NewScanner()
		_, _, err := scan.Scan([]byte(request))
		require.EqualError(t, err, ErrNoHost.Error())
	})

	t.Run("with content-length", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\nContent-Length: 13\r\n\r\nHello, world!rest"
		scan := NewScanner()
		_, _, err := scan.Scan([]byte(request))
		require.EqualError(t, err, ErrNoHost.Error())
	})

	t.Run("with content-length and host", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\nContent-Length: 13\r\nHost: www.google.com\r\n\r\nHello, world!rest"
		scan := NewScanner()
		host, endsAt, err := scan.Scan([]byte(request))
		require.NoError(t, err)
		require.Equal(t, "www.google.com", host)
		require.Equal(t, "rest", request[endsAt:])
	})
}
