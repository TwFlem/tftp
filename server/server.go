package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"tftp/packet"
	"time"
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
			n, targetAddr, err := conn.ReadFromUDP(req)
			if err != nil {
				fmt.Println("err reading request from connection:", err)
				continue
			}
			fmt.Println(n)
			fmt.Println(targetAddr)
			fmt.Println(string(req))

			op, err := packet.OpFrom(req)
			if err != nil {
				fmt.Println("err bad op from req:", err)
				continue
			}

			req, err := packet.RWRequestFrom(req)
			if err != nil {
				fmt.Println("err bad rw_rquest from req:", err)
				continue
			}

			switch op {
			case packet.OpWrite:
				fmt.Println("received write")
				err := handleWrite(targetAddr, req)
				if err != nil {
					fmt.Println("problem handling write:", err)
				}
			case packet.OpRead:
				fmt.Println("received read")
				err := handleRead(targetAddr, req)
				if err != nil {
					fmt.Println("problem handling read:", err)
				}
			default:
				fmt.Println("received bad op")
			}
		}
	}()

	select {
	case <-ctx.Done():
	}

	return nil
}

func handleWrite(peerAddr *net.UDPAddr, req packet.RWRequest) error {
	// choose random port
	hostAddrStr := fmt.Sprintf("%s:0", peerAddr.IP)
	hostAddr, err := net.ResolveUDPAddr("udp", hostAddrStr)
	if err != nil {
		return fmt.Errorf("problem resolving host response addr: %w", err)
	}
	conn, err := net.DialUDP("udp", peerAddr, hostAddr)
	if err != nil {
		return fmt.Errorf("problem dialing peer: %w", err)
	}
	defer conn.Close()

	return nil
}

func handleRead(peerAddr *net.UDPAddr, req packet.RWRequest) error {
	listenAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:")
	if err != nil {
		return fmt.Errorf("problem resolving listen addr for new read request: %w", err)
	}
	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("problem dialing peer: %w", err)
	}
	defer conn.Close()

	fp, err := os.Open(req.Filename)
	if err != nil {
		fmt.Errorf("problem opening file for reading: %w", err)
	}
	defer fp.Close()

	dataBuf := make([]byte, 512)
	ackBuf := make([]byte, 4)
	currTransmitBlock := 0
	retransmitInterval := time.Millisecond * 500
	timeout := time.Second * 5
	maxRetransmitAttempts := int(timeout / retransmitInterval)

	reachedEOF := false
	for !reachedEOF {
		n, err := io.ReadFull(fp, dataBuf)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			reachedEOF = true
		} else if err != nil {
			return fmt.Errorf("problem while reading file during transfer: %w", err)
		}

		currTransmitBlock++
		dataOut := packet.NewData(currTransmitBlock, dataBuf[:n])

		var retransmitAttempts int
		var lastAckBlock int
		for ; lastAckBlock != currTransmitBlock && retransmitAttempts < maxRetransmitAttempts; retransmitAttempts++ {
			n, err = conn.WriteToUDP(dataOut, peerAddr)
			if err != nil {
				return fmt.Errorf("problem while writing block=%d during transfer: %w", currTransmitBlock, err)
			}

			err = conn.SetReadDeadline(time.Now().Add(retransmitInterval))
			if err != nil {
				return fmt.Errorf("problem setting read deadline for retransmit internval of block=%d: %w", currTransmitBlock, err)
			}
			n, err = conn.Read(ackBuf)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}
				return fmt.Errorf("problem waiting for ack of block=%d during transfer: %w", currTransmitBlock, err)
			}

			lastAckBlock, err = packet.BlockFromAck(ackBuf)
			if err != nil {
				return fmt.Errorf("problem getting ack during transfer: %w", err)
			}
		}

		if retransmitAttempts >= maxRetransmitAttempts {
			if reachedEOF {
				fmt.Println("reached max retransmit attempts on last data packet, continuing assuming client received everything")
				break
			}
			return fmt.Errorf("max retransmit attempts reached for block=%d before EOF: %w", currTransmitBlock, err)
		}
	}

	return nil
}
