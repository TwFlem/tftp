package client

import (
	"bufio"
	"context"
	"errors"
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

	var serverAddr *net.UDPAddr
	var n int

	timeout := time.Second * 5
	blockTransferSize := 512
	tftpDataPacketSize := 2 + 2 + blockTransferSize
	dataTransferredBytes := 0
	dataPacketBuf := make([]byte, tftpDataPacketSize)

	err = conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return 0, fmt.Errorf("problem setting deadline read transfer: %w", err)
	}

	n, serverAddr, err = conn.ReadFromUDP(dataPacketBuf)
	if err != nil {
		return 0, fmt.Errorf("problem reading data from data response: %w", err)
	}

	data, err := packet.DataFrom(dataPacketBuf[:n])
	if err != nil {
		return 0, fmt.Errorf("problem marshaling data: %w", err)
	}

	if data.Block != 1 {
		return 0, fmt.Errorf("server did not start with first block: %w", err)
	}

	dataTransferredBytes += len(data.Payload)
	currDataSize := len(data.Payload)
	lastBlock := data.Block
	retransmitInterval := time.Millisecond * 500

	// TODO: figure out how better allow callers to consume this data and then remove.
	debug := []byte{}

	var retransmitAttempts int
	var maxRetransmitAttempts = int(timeout / retransmitInterval)
	for ; currDataSize == blockTransferSize && retransmitAttempts < maxRetransmitAttempts; retransmitAttempts++ {
		ack := packet.NewAck(lastBlock)
		_, err = conn.WriteToUDP(ack, serverAddr)
		if err != nil {
			return 0, fmt.Errorf("problem acking data from data response for block=%d: %w", data.Block, err)
		}

		err = conn.SetReadDeadline(time.Now().Add(retransmitInterval))
		if err != nil {
			return 0, fmt.Errorf("problem setting deadline read transfer: %w", err)
		}
		n, serverAddr, err = conn.ReadFromUDP(dataPacketBuf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			return 0, fmt.Errorf("problem reading data from data response: %w", err)
		}

		data, err := packet.DataFrom(dataPacketBuf[:n])
		if err != nil {
			return dataTransferredBytes, fmt.Errorf("problem marshaling data: %w", err)
		}

		if data.Block != lastBlock+1 {
			continue
		}
		retransmitAttempts = 0

		lastBlock = data.Block
		dataTransferredBytes += len(data.Payload)
		currDataSize = len(data.Payload)
		debug = append(debug, data.Payload...)

	}

	if currDataSize > blockTransferSize {
		return 0, fmt.Errorf("critical: somehow a payload size was larger than the agreed upon transfer size")
	}

	if retransmitAttempts >= maxRetransmitAttempts {
		return dataTransferredBytes, fmt.Errorf("max retransmit attempts reached: expectedBlock=%d lastReceivedBlock=%d %w", lastBlock+1, data.Block, err)
	}

	// This is a courtesy. At this point, the client got what it needed. We will not dally around to make sure the serving host received the final ack.
	ack := packet.NewAck(lastBlock)
	_, err = conn.WriteToUDP(ack, serverAddr)

	fmt.Println("bytes transferred", dataTransferredBytes)
	fmt.Println("last block", lastBlock)
	fmt.Println("data", string(debug))

	return 0, nil
}
