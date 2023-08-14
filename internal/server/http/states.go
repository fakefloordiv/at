package http

type serverState int

const (
	eAmass serverState = iota
	eTransit
)
