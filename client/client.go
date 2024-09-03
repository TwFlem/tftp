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

	dataTransferedBytes := 0
	tftpDataTransferSize := 512
	// op+block+tsize
	tftpDataPacketSize := 2 + 2 + tftpDataTransferSize
	dataBuf := make([]byte, tftpDataPacketSize)
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			return 0, fmt.Errorf("problem setting deadline read transfer: %w", err)
		}
		n, serverAddr, err := conn.ReadFromUDP(dataBuf)
		if err != nil {
			return 0, fmt.Errorf("problem reading data from data response: %w", err)
		}

		op, err := packet.OpFrom(dataBuf)
		if err != nil {
			return 0, fmt.Errorf("problem getting op from data response: %w", op, err)
		}
		if op != packet.OpData {
			return 0, fmt.Errorf("expected data response but got op=%s: %w", op, err)
		}

		block, err := packet.BlockFrom(dataBuf)
		if err != nil {
			return 0, fmt.Errorf("problem reading block from data response: %w", err)
		}

		data, err := packet.DataFrom(dataBuf[:n])
		if err != nil {
			return 0, fmt.Errorf("problem reading data from data response: %w", err)
		}
		dataTransferedBytes += len(data)

		fmt.Println("block:", block, "data:", string(data))

		ack := packet.NewAck(block)
		_, err = conn.WriteToUDP(ack, serverAddr)
		if err != nil {
			return 0, fmt.Errorf("problem acking data from data response for block=%d: %w", block, err)
		}

		fmt.Println(block, len(data))
		if len(data) < tftpDataTransferSize {
			fmt.Println("last packet received and sent out last ack")
			break
		}
	}

	fmt.Println("finished reading", dataTransferedBytes, "bytes from", dstFilename)
	return 0, nil
}
