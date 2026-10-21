package main

import (
    "fmt"
    "log"
    "time"
)

type TunnelClient struct {
    transport   *UDPTransport
    peer        string
    reassembler *Reassembler
}

func NewClient(listenAddr, peer string) *TunnelClient {
    tr, err := NewUDPTransport(listenAddr)
    if err != nil {
        log.Fatalf("client transport init failed: %v", err)
    }
    return &TunnelClient{transport: tr, peer: peer, reassembler: NewReassembler()}
}

func (c *TunnelClient) Connect() error {
    syn := BuildFrame(FrameTypeSyn, 1, 1, 0, 1, []byte("hello-server"), 0)
    return c.transport.SendTo(c.peer, syn)
}

func (c *TunnelClient) SendText(text string) error {
    parts := SplitPayload([]byte(text), 128)
    streamID := uint16(1)
    seq := uint32(2)
    for i, part := range parts {
        flags := uint8(0)
        if i == 0 {
            flags |= FlagFirst
        }
        if i == len(parts)-1 {
            flags |= FlagLast
        }

        frame := BuildFrame(FrameTypeData, streamID, seq, uint16(i), uint16(len(parts)), part, flags)
        if err := c.transport.SendTo(c.peer, frame); err != nil {
            return err
        }
        seq++
        time.Sleep(15 * time.Millisecond)
    }
    return nil
}

func (c *TunnelClient) Run() {
    for {
        raw, remote, err := c.transport.ReadFrom()
        if err != nil {
            log.Printf("client read error: %v", err)
            continue
        }
        if err := ValidateFrame(raw); err != nil {
            log.Printf("client invalid frame from %s: %v", remote, err)
            continue
        }

        h, payload, err := ParseFrame(raw)
        if err != nil {
            log.Printf("client parse error: %v", err)
            continue
        }

        log.Printf("client recv type=%d stream=%d seq=%d frag=%d/%d len=%d from=%s", h.Type, h.StreamID, h.Seq, h.FragIndex, h.FragTotal, len(payload), remote)

        switch h.Type {
        case FrameTypeAck:
            log.Printf("client acked: %s", string(payload))
        case FrameTypeData:
            c.reassembler.Add(h, payload)
            if assembled, ok := c.reassembler.Assemble(h.StreamID, h.Seq, h.FragTotal); ok {
                fmt.Printf("client assembled payload: %s\n", string(assembled))
            }
        case FrameTypePong:
            fmt.Printf("client got pong: %s\n", string(payload))
        }
    }
}
