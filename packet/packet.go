package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	maxFilenameSize  = 255
	opPacketByteSize = 2
)

var (
	errInvalidOp       = errors.New("invalid_op")
	errInvalidMode     = errors.New("invalid_mode")
	errMalformedPacket = errors.New("invalid_packet")
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

func OpFrom(packet []byte) (Op, error) {
	if len(packet) < 2 {
		return OpInvalid, fmt.Errorf("not enough bytes for an op code: %w", errInvalidOp)
	}

	op := binary.BigEndian.Uint16(packet[:2])
	if !(1 <= op && op <= 5) {
		return OpInvalid, fmt.Errorf("%d is not a valid op code: %w", op, errInvalidOp)
	}

	return Op(op), nil
}

type Mode string

const (
	ModeNetascii Mode = "netascii"
	ModeOctet    Mode = "octet"
)

// packet structure: op:2 - filename->0 - mode->0
func FromRWRequest(op Op, filename string, mode Mode) []byte {
	packet := make([]byte, opPacketByteSize+len(filename)+1+len(mode)+1)

	binary.BigEndian.PutUint16(packet, uint16(op))
	n := 2

	n += copy(packet[2:], filename)
	packet[n] = 0
	n++

	n += copy(packet[n:], mode)
	packet[n] = 0
	n++

	return packet
}

// RWRequest read and write requests payloads excluding the op code
type RWRequest struct {
	Filename string
	Mode     Mode
}

// packet structure: op:2 - filename->0 - mode->0
// RWRequestFrom Extracts the fields from either the read or write request. Aside from the op code,
// Read and Write requests contain the same data.
func RWRequestFrom(packet []byte) (RWRequest, error) {
	fields := bytes.Split(packet[2:], []byte{0})
	if len(fields) < 2 {
		return RWRequest{}, fmt.Errorf("missing filename or mode: %w", errMalformedPacket)
	}
	if len(fields[0]) < 1 {
		return RWRequest{}, fmt.Errorf("missing or invalid filename: %w", errMalformedPacket)
	}
	if len(fields[1]) < 1 {
		return RWRequest{}, fmt.Errorf("missing mode: %w", errMalformedPacket)
	}

	filename := string(fields[0])
	mode := Mode(string(fields[1]))
	if !(mode == ModeOctet || mode == ModeNetascii) {
		return RWRequest{}, fmt.Errorf("invalid mode: %w", errMalformedPacket)
	}

	return RWRequest{
		Mode:     mode,
		Filename: filename,
	}, nil
}
