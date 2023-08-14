package http1

import (
	"at/internal/scan"
	"bytes"
	"github.com/indigo-web/utils/uf"
)

var (
	_ scan.Scanner = NewScanner()

	hostKey          = []byte("host:")
	contentLengthKey = []byte("content-length:")
	// this variable must hold the value of the LONGEST key, including a colon at the end
	maxKeyLen = len(contentLengthKey)
)

type Scanner struct {
	contentLength int
	// TODO: to check, whether request's body is chunked, we need to parse
	//  the value of the Transfer-Encoding header
	isChunked       bool
	state           parserState
	headerKeyBuffer []byte
	hostValueBuffer []byte
	host            string
	chunkedScanner  *chunkedBodyScanner
}

func NewScanner() *Scanner {
	return &Scanner{
		headerKeyBuffer: make([]byte, 0, maxKeyLen),
		hostValueBuffer: make([]byte, 0, 4096),
		chunkedScanner:  newChunkedScanner(),
	}
}

func (s *Scanner) Scan(data []byte) (to string, endsAt int, err error) {
	var (
		pos             int
		originalDataLen = len(data)
	)

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
	case eBody:
		goto body
	default:
		panic("BUG: unknown scan state")
	}

requestLine:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		return "", -1, nil
	}

	data = data[pos+1:]
	s.state = eHeaderKey
	// no goto, as headerKey is anyway just below. Just let it fall through without any extra
	// instructions

headerKey:
	if len(data) == 0 {
		return s.host, -1, nil
	}

	switch data[0] {
	case '\r':
		data = data[1:]
		s.state = eHeaderKeyCR
		goto headerKeyCR
	case '\n':
		if len(s.host) == 0 {
			return "", -1, ErrNoHost
		}

		data = data[1:]
		s.state = eBody
		goto body
	}

	s.headerKeyBuffer = append(s.headerKeyBuffer, data[:maxKeyLen-len(s.headerKeyBuffer)]...)

	if len(s.headerKeyBuffer) >= len(hostKey) && s.headerKeyBuffer[len(hostKey)-1] == ':' {
		if !equalfold(s.headerKeyBuffer[:len(hostKey)], hostKey) {
			s.state = eOtherHeaderValue
			goto otherHeaderValue
		}

		data = data[len(hostKey):]
		s.state = eHostValue
		goto hostValue
	} else if len(s.headerKeyBuffer) >= len(contentLengthKey) {
		if !equalfold(s.headerKeyBuffer, contentLengthKey) {
			s.state = eOtherHeaderValue
			goto otherHeaderValue
		}

		data = data[len(contentLengthKey):]
		s.state = eContentLengthValue
		goto contentLengthValue
	}

	return s.host, -1, nil

headerKeyCR:
	if len(data) == 0 {
		return s.host, -1, nil
	}

	if data[0] != '\n' {
		return "", -1, ErrBadRequest
	}

	if len(s.hostValueBuffer) == 0 {
		return "", -1, ErrNoHost
	}

	data = data[1:]
	s.state = eBody
	goto body

otherHeaderValue:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		return s.host, -1, nil
	}

	data = data[pos+1:]
	s.headerKeyBuffer = s.headerKeyBuffer[:0]
	s.state = eHeaderKey
	goto headerKey

hostValue:
	pos = bytes.IndexByte(data, '\n')
	if pos == -1 {
		if len(s.hostValueBuffer)+len(data) > cap(s.hostValueBuffer) {
			return "", -1, ErrTooLong
		}

		s.hostValueBuffer = append(s.hostValueBuffer, data...)

		return "", -1, nil
	}

	{
		value := trimPrefixSpaces(data[:pos])
		if value[len(value)-1] == '\r' {
			value = value[:len(value)-1]
		}

		if len(s.hostValueBuffer)+len(value) > cap(s.hostValueBuffer) {
			return "", -1, ErrTooLong
		}

		s.hostValueBuffer = append(s.hostValueBuffer, value...)
		s.host = uf.B2S(s.hostValueBuffer)
		data = data[pos+1:]
		s.headerKeyBuffer = s.headerKeyBuffer[:0]
		s.state = eHeaderKey
		goto headerKey
	}

contentLengthValue:
	if len(data) == 0 {
		return s.host, -1, nil
	}

	for i, char := range data {
		if char == ' ' {
			continue
		}

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
		s.headerKeyBuffer = s.headerKeyBuffer[:0]
		s.state = eHeaderKey
		goto headerKey
	default:
		// content-length value is invalid in this case
		return "", -1, ErrBadRequest
	}

contentLengthValueCR:
	if len(data) == 0 {
		return s.host, -1, nil
	}

	if data[0] != '\n' {
		return "", -1, ErrBadRequest
	}

	data = data[1:]
	s.headerKeyBuffer = s.headerKeyBuffer[:0]
	s.state = eHeaderKey
	goto headerKey

body:
	if s.isChunked {
		endsAt, err = s.chunkedScanner.Parse(data)
		return s.host, originalDataLen - endsAt, err
	}

	if len(data) >= s.contentLength {
		return s.host, originalDataLen - len(data) + s.contentLength, nil
	}

	s.contentLength -= len(data)

	return s.host, -1, nil
}

func (s *Scanner) Release() {
	s.contentLength = 0
	s.hostValueBuffer = s.hostValueBuffer[:0]
	s.host = ""
	s.isChunked = false
	s.state = eRequestLine
}

func trimPrefixSpaces(b []byte) []byte {
	for i, char := range b {
		if char != ' ' {
			return b[i:]
		}
	}

	return b[:0]
}

func equalfold(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i]|0x20 != b[i] {
			return false
		}
	}

	return true
}
