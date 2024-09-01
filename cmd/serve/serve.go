package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"tftp/server"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Kill, os.Interrupt)
	defer cancel()

	s := server.New()
	err := s.StartAndListen(ctx)
	if err != nil {
		fmt.Println(err)
	}
}
