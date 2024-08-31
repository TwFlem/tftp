package main

import (
	"context"
	"fmt"
	"tftp/client"
)

func main() {
	ctx := context.Background()
	c := client.New("todo")
	bytesWritten, err := c.Write(ctx, "data/lorem.txt", "todo")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("finished reading %d bytes from %s\n", bytesWritten, "data/lorem.txt")
}
