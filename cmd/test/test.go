package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"tftp/client"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Kill, os.Interrupt)
	defer cancel()

	c := client.New("127.0.0.1:6969")
	n, err := c.Read(ctx, "data/lorem.txt")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("wrote n bytes", n)
}
