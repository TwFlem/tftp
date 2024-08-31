package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	fixedPacketSize = 512
)

type Client struct {
	destination string
	timeout     time.Duration
}

func New(destination string) *Client {
	return &Client{
		destination: destination,
		// TODO: configurable
		timeout: time.Second * 5,
	}
}

func (s Client) Write(ctx context.Context, srcFilename string, dstFilename string) (int, error) {
	fp, err := os.Open(srcFilename)
	if err != nil {
		return 0, fmt.Errorf("problem opening file %s: %w", srcFilename, err)
	}
	defer fp.Close()

	r := bufio.NewReader(fp)
	packet := make([]byte, fixedPacketSize)

	sizeTransfered := 0
	for {
		n, err := io.ReadFull(r, packet)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			sizeTransfered += n
			break
		}
		if err != nil {
			return sizeTransfered, fmt.Errorf("unexpected error while reading file: %w", err)
		}
		sizeTransfered += n
	}

	return sizeTransfered, nil
}
