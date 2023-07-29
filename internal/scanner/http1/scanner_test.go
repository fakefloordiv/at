package http1

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestScanner(t *testing.T) {
	t.Run("simple get", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\n\r\n"
		scan := NewScanner()
		report, done, rest, err := scan.Scan([]byte(request))
		require.NoError(t, err)
		require.Empty(t, rest)
		require.True(t, done)
		require.Empty(t, report.Receiver)
		require.Zero(t, report.ContentLength)
		require.False(t, report.IsChunked)
	})

	t.Run("with content-length", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\nContent-Length: 13\r\n\r\n"
		scan := NewScanner()
		report, done, rest, err := scan.Scan([]byte(request))
		require.NoError(t, err)
		require.Empty(t, rest)
		require.True(t, done)
		require.Empty(t, report.Receiver)
		require.Equal(t, 13, report.ContentLength)
		require.False(t, report.IsChunked)
	})

	t.Run("with content-length and host", func(t *testing.T) {
		request := "GET / HTTP/1.1\r\nContent-Length: 13\r\nHost: www.google.com\r\n\r\n"
		scan := NewScanner()
		report, done, rest, err := scan.Scan([]byte(request))
		require.NoError(t, err)
		require.Empty(t, rest)
		require.True(t, done)
		require.Equal(t, "www.google.com", report.Receiver)
		require.Equal(t, 13, report.ContentLength)
		require.False(t, report.IsChunked)
	})

	t.Run("lol", func(t *testing.T) {
		fmt.Println(strconv.Quote(string(generateRequest(5, "google.com", 13))))
	})
}
