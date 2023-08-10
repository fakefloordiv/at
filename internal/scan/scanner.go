package scan

type Report struct {
	Host          string
	ContentLength int
	IsChunked     bool
}

type Scanner interface {
	Scan(data []byte) (report Report, done bool, rest []byte, err error)
	Body(data []byte) (endsAt int, err error)
	Release()
}
