package packet

import (
	"fmt"
	"testing"
)

func TestWriteRequest_Pack(t *testing.T) {
	wr := WriteRequest{
		Mode:     ModeOctet,
		Filename: "/some/kind/of/file/name",
	}
	packet := wr.Pack()
	fmt.Println(len(packet), string(packet))
}
