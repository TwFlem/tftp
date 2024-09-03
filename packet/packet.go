package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	maxFilenameSize = 255
)

var (
	ErrFileNotFound = Error{
		Code:    1,
		Message: "file_not_found",
	}
	ErrAccessViolation = Error{
		Code:    2,
		Message: "access_violation",
	}
	ErrAllocationExceeded = Error{
		Code:    3,
		Message: "disk_full_or_allocation_exceeded",
	}
	ErrInvalidOperation = Error{
		Code:    4,
		Message: "invalid_tftp_operation",
	}
	// ErrInvalidTransferID invalid, unkown, or in use ports.
	ErrInvalidTransferID = Error{
		Code:    5,
		Message: "invalid_transfer_id",
	}
	ErrFileAlreadyExists = Error{
		Code:    6,
		Message: "file_already_exists",
	}
	ErrInvalidUser = Error{
		Code:    7,
		Message: "invalid_user",
	}
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
		return OpInvalid, fmt.Errorf("not enough bytes for an op code: %w", ErrInvalidOperation)
	}

	op := binary.BigEndian.Uint16(packet[:2])
	if !(1 <= op && op <= 5) {
		return OpInvalid, fmt.Errorf("%d is not a valid op code: %w", op, ErrInvalidOperation)
	}

	return Op(op), nil
}

type Mode string

const (
	ModeNetascii Mode = "netascii"
	ModeOctet    Mode = "octet"
)

// rwrequest packet structure: op:2 - filename->0 - mode->0
func NewRWRequest(op Op, filename string, mode Mode) []byte {
	packet := make([]byte, 2+len(filename)+1+len(mode)+1)

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

// rwrequest packet structure: op:2 - filename->0 - mode->0
// RWRequestFrom Extracts the fields from either the read or write request. Aside from the op code,
// Read and Write requests contain the same data.
func RWRequestFrom(packet []byte) (RWRequest, error) {
	fields := bytes.Split(packet[2:], []byte{0})
	if len(fields) < 2 {
		return RWRequest{}, Error{0, "missing_filename_or_mode"}
	}
	if len(fields[0]) < 1 {
		return RWRequest{}, Error{0, "missing_or_invalid_filename"}
	}
	if len(fields[1]) < 1 {
		return RWRequest{}, Error{0, "missing_or_invalid_mode"}
	}

	filename := string(fields[0])
	mode := Mode(string(fields[1]))
	if !(mode == ModeOctet || mode == ModeNetascii) {
		return RWRequest{}, Error{0, "missing_mode"}
	}

	return RWRequest{
		Mode:     mode,
		Filename: filename,
	}, nil
}

// data packet structure: op:2 - block:2 - data:tsize
// ack packet structure: op:2 - block:2
func BlockFrom(packet []byte) (int, error) {
	if len(packet) < 4 {
		return 0, Error{0, "missing_block_number"}
	}
	return int(binary.BigEndian.Uint16(packet[2:4])), nil
}

// data packet structure: op:2 - block:2 - data:tsize
func DataFrom(packet []byte) ([]byte, error) {
	if len(packet) < 4 {
		return nil, Error{0, "missing_data"}
	}
	return packet[4:], nil
}

// TODO: see what happens if we make this 0 allocation
// data packet structure: op:2 - block:2 - data:tsize
func NewData(block int, data []byte) []byte {
	packet := make([]byte, 2+2+len(data))

	binary.BigEndian.PutUint16(packet, uint16(3))
	n := 2

	binary.BigEndian.PutUint16(packet[n:], uint16(block))
	n += 2

	n += copy(packet[n:], data)

	return packet
}

// ack packet structure: op:2 - size:2 - data:size
func NewAck(block int) []byte {
	packet := make([]byte, 2+2)

	binary.BigEndian.PutUint16(packet, uint16(4))
	n := 2
	binary.BigEndian.PutUint16(packet[n:], uint16(block))

	return packet
}

type Error struct {
	Code    int
	Message string
}

func (e Error) String() string {
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}

func (e Error) Error() string {
	return e.String()
}

// error packet structure: op:2 - error_code:2 - message->0
func ErrorFrom(packet []byte) (Error, error) {
	if len(packet) < 4 {
		return Error{}, errors.New("malformed_packet")
	}

	code := int(binary.BigEndian.Uint16(packet[2:4]))
	if len(packet) < 5 {
		return Error{Code: code}, nil
	}

	sentinel := 4
	for ; sentinel < len(packet) && packet[sentinel] != 0; sentinel++ {
	}
	return Error{
		Code:    code,
		Message: string(packet[4:sentinel]),
	}, nil
}

// error packet structure: op:2 - error_code:2 - message->0
func NewError(err error) []byte {
	code := 0
	msg := err.Error()
	if pErr, ok := err.(Error); ok {
		code = pErr.Code
		msg = pErr.Message
	}
	msgBytes := []byte(msg)
	packet := make([]byte, 2+2+len(msgBytes)+1)

	binary.BigEndian.PutUint16(packet, uint16(OpError))
	n := 2
	binary.BigEndian.PutUint16(packet[n:], uint16(code))
	n += 2

	n += copy(packet[n:], msgBytes)
	packet[n] = 0
	n++

	return packet
}
