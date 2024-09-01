package server

import (
	"context"
	"fmt"
	"net"
)

type Server struct {
	conn *net.UDPConn
}

func New() Server {
	return Server{}
}

func (s Server) StartAndListen(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6969")
	if err != nil {
		return fmt.Errorf("problem resolving udp address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("problem starting udp connection: %w", err)
	}
	defer conn.Close()
	s.conn = conn

	go func() {
		// Should be more than enough for read and write requests
		req := make([]byte, 512)
		for {
			fmt.Println("trying to read from thing")
			n, addr, err := conn.ReadFromUDP(req)
			if err != nil {
				fmt.Println("err reading request from connection:", err)
			}
			fmt.Println(n)
			fmt.Println(addr)
			fmt.Println(string(req))
		}
	}()

	select {
	case <-ctx.Done():
	}

	return nil
}
