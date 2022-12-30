package main

import "fmt"

type parameters struct {
	host string
	port int
	//staticDir string
	//flag      bool
}

func (p *parameters) getHttpAddress() string {
	addr := p.host
	if addr == "" {
		addr = "127.0.0.1"
	}

	if p.port > 0 {
		addr = fmt.Sprintf("%s:%d", addr, p.port)
	}

	return addr
}
