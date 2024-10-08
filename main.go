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

	c := client.New("todo")
	bytesWritten, err := c.Write(ctx, "data/lorem.txt", "todo")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("finished reading %d bytes from %s\n", bytesWritten, "data/lorem.txt")
}
