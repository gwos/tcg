package tools

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// InitPacket is used during the handshake with the client
type InitPacket struct {
	// initialization vector is a 128 bytes array
	Iv        []byte
	Timestamp uint32
}

// NewInitPacket initialize an InitPacket by creating a random initialization
// vector and filling the timestamp attribute with the current epoch time
func NewInitPacket() (*InitPacket, error) {
	initPacket := InitPacket{Iv: make([]byte, 128), Timestamp: 0}
	var err error
	_, err = rand.Read(initPacket.Iv)
	initPacket.Timestamp = uint32(time.Now().Unix())
	return &initPacket, err
}

// Write writes the current InitPacket to an io.Writer such as a TCPConnection
func (p *InitPacket) Write(w io.Writer) error {
	// Transforming to network bytes
	packet := new(bytes.Buffer)
	_ = binary.Write(packet, binary.BigEndian, p.Iv)
	_ = binary.Write(packet, binary.BigEndian, p.Timestamp)
	b := packet.Bytes()
	n, err := w.Write(b)
	// Consistency check
	if err == nil && n != len(b) {
		err = fmt.Errorf("%d bytes written but the packet is %d bytes", n, len(b))
	}
	return err
}
