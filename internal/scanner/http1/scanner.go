package http1

import (
	"atmen/internal/scanner"
	"bytes"
	"errors"
	"github.com/indigo-web/utils/uf"
)

var (
	hostKey          = []byte("host")
	contentLengthKey = []byte("content-length")
)

type Scanner struct {
	contentLength int
	// TODO: to check, whether request's body is chunked, we need to parse
	//  the value of the Transfer-Encoding header
	isChunked       bool
	state           parserState
	headerKeyBuffer []byte
	hostValueBuffer []byte
}

func NewScanner() *Scanner {
	return &Scanner{
		headerKeyBuffer: make([]byte, 0, 100),
		hostValueBuffer: make([]byte, 0, 4096),
	}
}

func (s *Scanner) Scan(data []byte) (report scanner.Report, done bool, rest []byte, err error) {
	var pos int

	switch s.state {
	case eRequestLine:
		goto requestLine
	case eHeaderKey:
		goto headerKey
	case eHostValue:
		goto hostValue
	case eContentLengthValue:
		goto contentLengthValue
	case eContentLengthValueCR:
		goto contentLengthValueCR
	case eOtherHeaderValue:
		goto otherHeaderValue
	case ePostHeaderValue:
		goto postHeaderValue
	}

requestLine:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		return report, false, data[:0], nil
	}

	data = data[pos+1:]
	s.state = eHeaderKey

headerKey:
	if len(data) == 0 {
		return report, false, data, nil
	}

	switch data[0] {
	case '\r':
		data = data[1:]
		s.state = eHeaderKeyCR
		goto headerKeyCR
	case '\n':
		s.state = eRequestLine

		return scanner.Report{
			Receiver:      uf.B2S(s.hostValueBuffer),
			ContentLength: s.contentLength,
			IsChunked:     false,
		}, true, data[1:], nil
	}

	pos = bytes.IndexByte(data, ':')
	if pos == -1 {
		if len(s.headerKeyBuffer)+len(data) > cap(s.headerKeyBuffer) {
			return report, true, nil, errors.New("header key is too long")
		}

		s.headerKeyBuffer = append(s.headerKeyBuffer, data...)

		return report, false, data[:0], nil
	}

	s.headerKeyBuffer = append(s.headerKeyBuffer, data[:pos]...)
	data = trimSuffixSpaces(data[pos+1:])

	switch {
	case bytes.EqualFold(s.headerKeyBuffer, hostKey):
		s.state = eHostValue
		goto hostValue
	case bytes.EqualFold(s.headerKeyBuffer, contentLengthKey):
		s.state = eContentLengthValue
		goto contentLengthValue
	default:
		s.state = eOtherHeaderValue
		goto otherHeaderValue
	}

headerKeyCR:
	if len(data) == 0 {
		return report, false, data, nil
	}

	if data[0] != '\n' {
		return report, true, nil, errors.New("incomplete CRLF sequence")
	}

	s.state = eRequestLine

	return scanner.Report{
		Receiver:      uf.B2S(s.hostValueBuffer),
		ContentLength: s.contentLength,
		IsChunked:     false,
	}, true, data[1:], nil

otherHeaderValue:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		return report, false, data[:0], nil
	}

	data = data[pos+1:]
	s.state = ePostHeaderValue
	goto postHeaderValue

hostValue:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		if len(s.hostValueBuffer)+len(data) > cap(s.hostValueBuffer) {
			return report, true, nil, errors.New("host is too long")
		}

		s.hostValueBuffer = append(s.hostValueBuffer, data...)

		return report, false, data[:0], nil
	}

	{
		value := data[:pos]
		if value[len(value)-1] == '\r' {
			value = value[:len(value)-1]
		}

		if len(s.hostValueBuffer)+len(value) > cap(s.hostValueBuffer) {
			return report, true, nil, errors.New("host is too long")
		}

		s.hostValueBuffer = append(s.hostValueBuffer, value...)
		data = data[pos+1:]
		s.state = ePostHeaderValue
		goto postHeaderValue
	}

contentLengthValue:
	if len(data) == 0 {
		return report, false, data, nil
	}

	for i, char := range data {
		if char < '0' || char > '9' {
			data = data[i:]
			break
		}

		s.contentLength = s.contentLength*10 + int(char) - '0'
	}

	switch data[0] {
	case '\r':
		data = data[1:]
		s.state = eContentLengthValueCR
		goto contentLengthValueCR
	case '\n':
		data = data[1:]
		s.state = ePostHeaderValue
		goto postHeaderValue
	default:
		return report, true, nil, errors.New("bad content-length value")
	}

contentLengthValueCR:
	if len(data) == 0 {
		return report, false, data, nil
	}

	if data[0] != '\n' {
		return report, true, nil, errors.New("incomplete CRLF sequence")
	}

	data = data[1:]
	s.state = ePostHeaderValue
	goto postHeaderValue

postHeaderValue:
	s.headerKeyBuffer = s.headerKeyBuffer[:0]
	s.state = eHeaderKey
	goto headerKey
}

func trimSuffixSpaces(b []byte) []byte {
	for i, char := range b {
		if char != ' ' {
			return b[i:]
		}
	}

	return b[:0]
}
