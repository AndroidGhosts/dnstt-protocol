package main

import (
    "fmt"
    "log"
    "time"
)

type Server struct {
    Transport *DNSTransport
    Reassemblers map[uint16]*StreamReassembler
}

func NewServer(listenAddr string) (*Server, error) {
    tr, err := NewDNSTransport(listenAddr)
    if err != nil {
        return nil, err
    }
    return &Server{
        Transport:    tr,
        Reassemblers: make(map[uint16]*StreamReassembler),
    }, nil
}

func (s *Server) Run() {
    for {
        raw, remote, err := s.Transport.ReadFrom()
        if err != nil {
            log.Println("server read error:", err)
            continue
        }
        if err := ValidateFrame(raw); err != nil {
            log.Println("server invalid frame:", err)
            continue
        }
        header, payload, err := ParseFrame(raw)
        if err != nil {
            log.Println("server parse error:", err)
            continue
        }
        log.Printf("server got type=%d stream=%d seq=%d frag=%d/%d len=%d from=%s", header.Type, header.StreamID, header.Seq, header.FragIndex, header.FragTotal, len(payload), remote)

        switch header.Type {
        case FrameTypeSyn:
            if _, ok := s.Reassemblers[header.StreamID]; !ok {
                s.Reassemblers[header.StreamID] = NewStreamReassembler()
            }
            ack, _ := BuildFrame(FrameTypeAck, header.StreamID, header.Seq, 0, 1, []byte("syn-ack"), 0)
            if err := s.Transport.SendTo(remote, ack); err != nil {
                log.Println("server send ack error:", err)
            }
        case FrameTypeData:
            if _, ok := s.Reassemblers[header.StreamID]; !ok {
                s.Reassemblers[header.StreamID] = NewStreamReassembler()
            }
            if err := s.Reassemblers[header.StreamID].Add(header, payload); err != nil {
                log.Println("server add fragment error:", err)
                continue
            }
            if assembled, ok := s.Reassemblers[header.StreamID].Assemble(header.Seq, header.FragTotal); ok {
                fmt.Printf("server assembled payload: %s\n", string(assembled))
                ack, _ := BuildFrame(FrameTypeAck, header.StreamID, header.Seq, 0, 1, []byte("ok"), 0)
                if err := s.Transport.SendTo(remote, ack); err != nil {
                    log.Println("ack send error:", err)
                }
            }
        case FrameTypePing:
            reply, _ := BuildFrame(FrameTypePong, header.StreamID, header.Seq+1, 0, 1, []byte("pong"), 0)
            if err := s.Transport.SendTo(remote, reply); err != nil {
                log.Println("pong send err:", err)
            }
        case FrameTypePong:
            log.Println("server received pong from client")
        }
    }
}

func (s *Server) SendTextTo(peer string, text string) error {
    streamID := uint16(1)
    seq := uint32(1)
    parts := SplitPayload([]byte(text), 128)
    for i, part := range parts {
        flags := uint8(0)
        if i == 0 {
            flags |= FlagFirst
        }
        if i == len(parts)-1 {
            flags |= FlagLast
        }
        frame, err := BuildFrame(FrameTypeData, streamID, seq, uint16(i), uint16(len(parts)), part, flags)
        if err != nil {
            return err
        }
        if err := s.Transport.SendTo(peer, frame); err != nil {
            return err
        }
        seq++
        time.Sleep(10 * time.Millisecond)
    }
    return nil
}

type Client struct {
    Transport *DNSTransport
    PeerAddr  string
    Reassemblers map[uint16]*StreamReassembler
}

func NewClient(listenAddr, peerAddr string) (*Client, error) {
    tr, err := NewDNSTransport(listenAddr)
    if err != nil {
        return nil, err
    }
    return &Client{
        Transport:    tr,
        PeerAddr:     peerAddr,
        Reassemblers: make(map[uint16]*StreamReassembler),
    }, nil
}

func (c *Client) Connect() error {
    syn, err := BuildFrame(FrameTypeSyn, 1, 1, 0, 1, []byte("hello-server"), 0)
    if err != nil {
        return err
    }
    return c.Transport.SendTo(c.PeerAddr, syn)
}

func (c *Client) Run() {
    go func() {
        for {
            time.Sleep(2 * time.Second)
            ping, err := BuildFrame(FrameTypePing, 1, uint32(time.Now().UnixNano()), 0, 1, []byte("ping"), 0)
            if err != nil {
                log.Println("client ping build error:", err)
                continue
            }
            if err := c.Transport.SendTo(c.PeerAddr, ping); err != nil {
                log.Println("client ping send error:", err)
            }
        }
    }()

    for {
        raw, remote, err := c.Transport.ReadFrom()
        if err != nil {
            log.Println("client read error:", err)
            continue
        }
        if err := ValidateFrame(raw); err != nil {
            log.Println("client invalid frame:", err)
            continue
        }
        header, payload, err := ParseFrame(raw)
        if err != nil {
            log.Println("client parse error:", err)
            continue
        }
        log.Printf("client got type=%d stream=%d seq=%d frag=%d/%d len=%d from=%s", header.Type, header.StreamID, header.Seq, header.FragIndex, header.FragTotal, len(payload), remote)

        switch header.Type {
        case FrameTypeAck:
            log.Println("client got ack")
        case FrameTypeData:
            if _, ok := c.Reassemblers[header.StreamID]; !ok {
                c.Reassemblers[header.StreamID] = NewStreamReassembler()
            }
            if err := c.Reassemblers[header.StreamID].Add(header, payload); err != nil {
                log.Println("client add fragment error:", err)
                continue
            }
            if assembled, ok := c.Reassemblers[header.StreamID].Assemble(header.Seq, header.FragTotal); ok {
                fmt.Printf("client assembled payload: %s\n", string(assembled))
            }
        case FrameTypePong:
            log.Printf("client got pong payload=%s", string(payload))
        }
    }
}
