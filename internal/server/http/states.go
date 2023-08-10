package http

type serverState int

const (
	eHeaders serverState = iota
	eBody
)
