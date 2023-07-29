package scanner

type Report struct {
	Receiver      string
	ContentLength int
	IsChunked     bool
}

type Scanner interface {
	Scan(data []byte) (report Report, done bool, rest []byte, err error)
}
