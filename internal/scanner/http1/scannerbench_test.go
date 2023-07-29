package http1

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkScanner(b *testing.B) {
	b.Run("simple get", func(b *testing.B) {
		scan := NewScanner()
		request := []byte("GET / HTTP/1.1\r\n\r\n")
		b.SetBytes(int64(len(request)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, _, _ = scan.Scan(request)
		}
	})

	b.Run("get with 5 headers", func(b *testing.B) {
		scan := NewScanner()
		request := generateRequest(5, "www.google.com", 13)
		b.SetBytes(int64(len(request)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, _, _ = scan.Scan(request)
		}
	})

	b.Run("get with 10 headers", func(b *testing.B) {
		scan := NewScanner()
		request := generateRequest(10, "www.google.com", 13)
		b.SetBytes(int64(len(request)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, _, _ = scan.Scan(request)
		}
	})

	b.Run("get with 50 headers", func(b *testing.B) {
		scan := NewScanner()
		request := generateRequest(10, "www.google.com", 13)
		b.SetBytes(int64(len(request)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, _, _ = scan.Scan(request)
		}
	})
}

func generateRequest(headersNum int, hostValue string, contentLengthValue int) (request []byte) {
	request = append(request,
		"GET /"+strings.Repeat("a", 500)+"\r\n"...,
	)

	for i := 0; i < headersNum; i++ {
		request = append(request,
			"some-random-header-name-nobody-cares-about"+strconv.Itoa(i)+": "+
				strings.Repeat("b", 100)+"\r\n"...,
		)
	}

	request = append(request, "Host: "+hostValue+"\r\n"...)
	request = append(request, "Content-Length: "+strconv.Itoa(contentLengthValue)+"\r\n"...)

	return append(request, '\r', '\n')
}
