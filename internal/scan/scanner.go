package scan

type Scanner interface {
	Scan(data []byte) (to string, endsAt int, err error)
	Release()
}
