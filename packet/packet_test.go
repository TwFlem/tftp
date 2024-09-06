package packet

import (
	"errors"
	"strings"
	"testing"
)

func TestRWRequest(t *testing.T) {
	type testCase struct {
		op       Op
		filename string
		mode     Mode
	}

	testCases := []testCase{
		{OpWrite, "/some/kind/of/file/name", ModeOctet},
		{OpRead, "/", ModeOctet},
		{OpWrite, "/other", ModeNetascii},
		{OpRead, "/0_0", ModeNetascii},
	}
	for _, tc := range testCases {
		packet := NewRWRequest(tc.op, tc.filename, tc.mode)
		op, err := OpFrom(packet)
		if err != nil {
			t.Fatal("problem when getting op from packet: ", err)
		}
		if op != tc.op {
			t.Fatal("expected write op but got ", op)
		}
		wr, err := RWRequestFrom(packet)
		if err != nil {
			t.Fatal(err)
		}
		if wr.Filename != tc.filename {
			t.Fatalf("filename: expected=\"%s\" actual=\"%s\"", tc.filename, wr.Filename)
		}
		if wr.Mode != tc.mode {
			t.Fatalf("mode: expected=\"%s\" actual=\"%s\"", tc.mode, wr.Mode)
		}
	}
}

func TestData(t *testing.T) {
	type testCase struct {
		block int
		data  []byte
	}

	testCases := []testCase{
		{0, []byte{0x1, 0x2, 0x3, 0x4, 0x5}},
		{1, []byte{0x1}},
		{2, []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}},
	}
	for _, tc := range testCases {
		packet := NewData(tc.block, tc.data)
		op, err := OpFrom(packet)
		if err != nil {
			t.Fatal("problem when getting op from packet:", err)
		}
		if op != OpData {
			t.Fatal("expected data op but got ", op)
		}
		data, err := DataFrom(packet)
		if err != nil {
			t.Fatal("problem when getting data from packet:", err)
		}
		if err != nil {
			t.Fatal("problem when getting block from packet:", err)
		}
		if data.Block != tc.block {
			t.Fatalf("block: expected=%d actual=%d", tc.block, data.Block)
		}
		if len(data.Payload) != len(tc.data) {
			t.Fatalf("data: expected=%v actual=%v", tc.data, data)
		}
		for i := range data.Payload {
			if data.Payload[i] != tc.data[i] {
				t.Fatalf("data: expected=%v actual=%v", tc.data, data)
			}
		}
	}
}

func TestAck(t *testing.T) {
	type testCase struct {
		block int
	}

	testCases := []testCase{
		{0},
		{1},
		{2},
	}
	for _, tc := range testCases {
		packet := NewAck(tc.block)
		op, err := OpFrom(packet)
		if err != nil {
			t.Fatal("problem when getting op from packet:", err)
		}
		if op != OpAck {
			t.Fatal("expected ack op but got ", op)
		}
		block, err := BlockFrom(packet)
		if err != nil {
			t.Fatal("problem when getting block from packet:", err)
		}
		if block != tc.block {
			t.Fatalf("block: expected=%d actual=%d", tc.block, block)
		}
	}
}

func TestError(t *testing.T) {
	type testCase struct {
		expectedCode int
		err          error
	}

	testCases := []testCase{
		{0, errors.New("something_unexpected")},
		{1, ErrFileNotFound},
		{2, ErrAccessViolation},
	}
	for _, tc := range testCases {
		packet := NewError(tc.err)
		op, err := OpFrom(packet)
		if err != nil {
			t.Fatal("problem when getting op from packet:", err)
		}
		if op != OpError {
			t.Fatal("expected error op but got ", op)
		}
		pErr, err := ErrorFrom(packet)
		if err != nil {
			t.Fatal("problem when getting error from packet:", err)
		}
		if pErr.Code != tc.expectedCode {
			t.Fatalf("code: expected=%d actual=%d", tc.expectedCode, pErr.Code)
		}
		if !strings.Contains(tc.err.Error(), pErr.Message) {
			t.Fatalf("message: expected=%d actual=%d", tc.expectedCode, pErr.Code)
		}
	}
}
