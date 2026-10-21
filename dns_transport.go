package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

const (
	ProtoMagic = 0x44534E54 // DSNT
)

const (
	FrameTypeData = 1
	FrameTypeAck  = 2
	FrameTypeSyn  = 3
	FrameTypeFin  = 4
	FrameTypePing = 5
	FrameTypePong = 6
)

const (
	FlagFirst = 1 << 0
	FlagLast  = 1 << 1
)

type FrameHeader struct {
	Magic      uint32
	Type       uint8
	Flags      uint8
	StreamID   uint16
	Seq        uint32
	FragIndex  uint16
	FragTotal  uint16
	PayloadLen uint16
	Checksum   uint32
}

func (h FrameHeader) Encode() []byte {
	b := make([]byte, 22)
	binary.BigEndian.PutUint32(b[0:4], h.Magic)
	b[4] = h.Type
	b[5] = h.Flags
	binary.BigEndian.PutUint16(b[6:8], h.StreamID)
	binary.BigEndian.PutUint32(b[8:12], h.Seq)
	binary.BigEndian.PutUint16(b[12:14], h.FragIndex)
	binary.BigEndian.PutUint16(b[14:16], h.FragTotal)
	binary.BigEndian.PutUint16(b[16:18], h.PayloadLen)
	binary.BigEndian.PutUint32(b[18:22], h.Checksum)
	return b
}

func DecodeFrameHeader(raw []byte) (FrameHeader, error) {
	if len(raw) < 22 {
		return FrameHeader{}, errors.New("frame too short")
	}
	h := FrameHeader{}
	h.Magic = binary.BigEndian.Uint32(raw[0:4])
	h.Type = raw[4]
	h.Flags = raw[5]
	h.StreamID = binary.BigEndian.Uint16(raw[6:8])
	h.Seq = binary.BigEndian.Uint32(raw[8:12])
	h.FragIndex = binary.BigEndian.Uint16(raw[12:14])
	h.FragTotal = binary.BigEndian.Uint16(raw[14:16])
	h.PayloadLen = binary.BigEndian.Uint16(raw[16:18])
	h.Checksum = binary.BigEndian.Uint32(raw[18:22])
	return h, nil
}

func BuildFrame(frameType uint8, streamID uint16, seq uint32, fragIndex, fragTotal uint16, payload []byte, flags uint8) ([]byte, error) {
	h := FrameHeader{
		Magic:      ProtoMagic,
		Type:       frameType,
		Flags:      flags,
		StreamID:   streamID,
		Seq:        seq,
		FragIndex:  fragIndex,
		FragTotal:  fragTotal,
		PayloadLen: uint16(len(payload)),
	}

	head := make([]byte, 18)
	binary.BigEndian.PutUint32(head[0:4], h.Magic)
	head[4] = h.Type
	head[5] = h.Flags
	binary.BigEndian.PutUint16(head[6:8], h.StreamID)
	binary.BigEndian.PutUint32(head[8:12], h.Seq)
	binary.BigEndian.PutUint16(head[12:14], h.FragIndex)
	binary.BigEndian.PutUint16(head[14:16], h.FragTotal)
	binary.BigEndian.PutUint16(head[16:18], h.PayloadLen)

	h.Checksum = crc32.ChecksumIEEE(append(head, payload...))
	buf := h.Encode()
	buf = append(buf, payload...)
	return buf, nil
}

func ParseFrame(raw []byte) (FrameHeader, []byte, error) {
	if len(raw) < 22 {
		return FrameHeader{}, nil, errors.New("frame too short")
	}
	h, err := DecodeFrameHeader(raw)
	if err != nil {
		return FrameHeader{}, nil, err
	}
	if h.Magic != ProtoMagic {
		return FrameHeader{}, nil, fmt.Errorf("bad magic: 0x%x", h.Magic)
	}
	if len(raw) < int(22+h.PayloadLen) {
		return FrameHeader{}, nil, errors.New("payload truncated")
	}
	payload := raw[22 : 22+h.PayloadLen]
	return h, payload, nil
}

func ValidateFrame(raw []byte) error {
	h, payload, err := ParseFrame(raw)
	if err != nil {
		return err
	}
	head := make([]byte, 18)
	binary.BigEndian.PutUint32(head[0:4], h.Magic)
	head[4] = h.Type
	head[5] = h.Flags
	binary.BigEndian.PutUint16(head[6:8], h.StreamID)
	binary.BigEndian.PutUint32(head[8:12], h.Seq)
	binary.BigEndian.PutUint16(head[12:14], h.FragIndex)
	binary.BigEndian.PutUint16(head[14:16], h.FragTotal)
	binary.BigEndian.PutUint16(head[16:18], h.PayloadLen)
	expected := crc32.ChecksumIEEE(append(head, payload...))
	if h.Checksum != expected {
		return fmt.Errorf("checksum mismatch: got=%d expected=%d", h.Checksum, expected)
	}
	return nil
}

type Reassembler struct {
	fragments map[uint16]map[uint32]map[uint16][]byte
}

func NewReassembler() *Reassembler {
	return &Reassembler{fragments: make(map[uint16]map[uint32]map[uint16][]byte)}
}

func (r *Reassembler) Add(h FrameHeader, payload []byte) error {
	if _, ok := r.fragments[h.StreamID]; !ok {
		r.fragments[h.StreamID] = make(map[uint32]map[uint16][]byte)
	}
	if _, ok := r.fragments[h.StreamID][h.Seq]; !ok {
		r.fragments[h.StreamID][h.Seq] = make(map[uint16][]byte)
	}
	r.fragments[h.StreamID][h.Seq][h.FragIndex] = append([]byte(nil), payload...)
	return nil
}

func (r *Reassembler) Assemble(streamID uint16, seq uint32, fragTotal uint16) ([]byte, bool) {
	streamMap, ok := r.fragments[streamID]
	if !ok {
		return nil, false
	}
	fragMap, ok := streamMap[seq]
	if !ok {
		return nil, false
	}
	if fragTotal == 0 {
		return nil, false
	}
	out := make([]byte, 0, 4096)
	for i := uint16(0); i < fragTotal; i++ {
		part, ok := fragMap[i]
		if !ok {
			return nil, false
		}
		out = append(out, part...)
	}
	delete(streamMap, seq)
	if len(streamMap) == 0 {
		delete(r.fragments, streamID)
	}
	return out, true
}

func SplitPayload(payload []byte, segmentSize int) [][]byte {
	if len(payload) == 0 {
		return nil
	}
	if segmentSize <= 0 {
		return [][]byte{payload}
	}
	parts := make([][]byte, 0, (len(payload)+segmentSize-1)/segmentSize)
	for i := 0; i < len(payload); i += segmentSize {
		end := i + segmentSize
		if end > len(payload) {
			end = len(payload)
		}
		parts = append(parts, payload[i:end])
	}
	return parts
}
