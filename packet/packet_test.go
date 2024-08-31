package packet

import (
	"testing"
)

func TestRequest_Packing(t *testing.T) {
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
		packet := FromRWRequest(tc.op, tc.filename, tc.mode)
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
