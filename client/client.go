package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"tftp/packet"
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

func (s Client) Write(ctx context.Context, dstFilename string, srcFilename string) (int, error) {
	fp, err := os.Open(srcFilename)
	if err != nil {
		return 0, fmt.Errorf("problem opening file %s: %w", srcFilename, err)
	}
	defer fp.Close()

	r := bufio.NewReader(fp)
	packet := make([]byte, fixedPacketSize)

	dstAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6969")
	if err != nil {
		return 0, fmt.Errorf("problem resolving UDP address: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, dstAddr)
	if err != nil {
		return 0, fmt.Errorf("problem dialing UDP address: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("this is a udp message"))
	if err != nil {
		return 0, fmt.Errorf("problem making write UDP request: %w", err)
	}

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

func (s Client) Read(ctx context.Context, dstFilename string) (int, error) {
	listenAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:")
	if err != nil {
		return 0, fmt.Errorf("problem resolving listening addr: %w", err)
	}
	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return 0, fmt.Errorf("problem establishing listening connection: %w", err)
	}
	defer conn.Close()

	initRequestAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:6969")
	if err != nil {
		return 0, fmt.Errorf("problem resolving UDP address: %w", err)
	}

	req := packet.NewRWRequest(packet.OpRead, "data/lorem.txt", packet.ModeOctet)
	_, err = conn.WriteToUDP(req, initRequestAddr)
	if err != nil {
		return 0, fmt.Errorf("problem writing read request to server: %w", err)
	}

	dataTransferredBytes := 0
	blockTransferSize := 512
	// op+block+tsize
	tftpDataPacketSize := 2 + 2 + blockTransferSize
	dataPacketBuf := make([]byte, tftpDataPacketSize)
	lastestBlock := 0
	debug := []byte{}
	lastestPayloadSize := blockTransferSize

	err = conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	if err != nil {
		return 0, fmt.Errorf("problem setting deadline read transfer: %w", err)
	}

	for lastestPayloadSize >= blockTransferSize {
		n, serverAddr, err := conn.ReadFromUDP(dataPacketBuf)
		if err != nil {
			return 0, fmt.Errorf("problem reading data from data response: %w", err)
		}

		data, err := packet.DataFrom(dataPacketBuf[:n])
		if err != nil {
			return dataTransferredBytes, fmt.Errorf("problem marshaling data: %w", err)
		}

		ack := packet.NewAck(data.Block)
		_, err = conn.WriteToUDP(ack, serverAddr)
		if err != nil {
			return 0, fmt.Errorf("problem acking data from data response for block=%d: %w", data.Block, err)
		}

		if data.Block == lastestBlock {
			continue
		}

		lastestBlock = data.Block
		dataTransferredBytes += len(data.Payload)
		// TODO: figure out how to allow something to consume this
		debug = append(debug, data.Payload...)
		lastestPayloadSize = len(data.Payload)

		// We need to time out eventually if we're pinging the same data back and forth
		err = conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			return 0, fmt.Errorf("problem setting deadline read transfer: %w", err)
		}
	}

	if lastestPayloadSize > blockTransferSize {
		return 0, fmt.Errorf("critical: somehow a payload size was larget than the agreed upon transfer size")
	}

	fmt.Println("bytes transferred", dataTransferredBytes)
	fmt.Println("last block", lastestBlock)
	fmt.Println("data", string(debug))

	return 0, nil
}
