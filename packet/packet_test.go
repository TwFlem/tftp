package packet

import (
	"testing"
)

func TestRequest_Packing(t *testing.T) {
	type testCase struct {
		filename string
		mode     Mode
	}

	testCases := []testCase{
		{"/some/kind/of/file/name", ModeOctet},
		{"/", ModeOctet},
		{"/other", ModeNetascii},
		{"/0_0", ModeNetascii},
	}
	for _, tc := range testCases {
		packet := FromWriteRequest(tc.filename, tc.mode)
		op, err := OpFrom(packet)
		if err != nil {
			t.Fatal(err)
		}
		if op != OpWrite {
			t.Fatal("expected write op but got ", op)
		}
		wr, err := WRRequestFrom(packet)
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
