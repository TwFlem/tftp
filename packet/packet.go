package packet

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	maxFilenameSize  = 255
	opPacketByteSize = 2
)

var (
	ErrInvalidOp = errors.New("invalid_op")
)

type Op int

const (
	OpInvalid Op = 0
	OpRead    Op = 1
	OpWrite   Op = 2
	OpData    Op = 3
	OpAck     Op = 4
	OpError   Op = 5
)

func (o Op) String() string {
	switch o {
	case 1:
		return "Read"
	case 2:
		return "Write"
	case 3:
		return "Data"
	case 4:
		return "Ack"
	case 5:
		return "Error"
	default:
		return "Invalid"
	}
}

func OpFromBytes(buf []byte) (Op, error) {
	if len(buf) < 2 {
		return OpInvalid, fmt.Errorf("not enough bytes for an op code: %w", ErrInvalidOp)
	}

	if buf[0] != '0' {
		return OpInvalid, fmt.Errorf("%b%b: %w", buf[0], buf[1], ErrInvalidOp)
	}

	switch buf[1] {
	case '1':
		return OpRead, nil
	case '2':
		return OpWrite, nil
	case '3':
		return OpData, nil
	case '4':
		return OpAck, nil
	case '5':
		return OpError, nil
	default:
		return OpInvalid, fmt.Errorf("%b%b: %w", buf[0], buf[1], ErrInvalidOp)
	}
}

type Mode string

const (
	ModeNetascii Mode = "netascii"
	ModeOctet    Mode = "octet"
)

type ReadRequest struct {
	Mode     Mode
	Filename string
}

type WriteRequest struct {
	Mode     Mode
	Filename string
}

// packet structure: op:2 - filename->0 - mode->0
func (wr WriteRequest) Pack() []byte {
	packet := make([]byte, opPacketByteSize+len(wr.Filename)+1+len(wr.Mode)+1)

	binary.BigEndian.PutUint16(packet, 2)
	n := 2

	n += copy(packet[2:], wr.Filename)
	packet[n] = '0'
	n++

	n += copy(packet[n:], wr.Mode)
	packet[n] = '0'
	n++

	return packet
}
