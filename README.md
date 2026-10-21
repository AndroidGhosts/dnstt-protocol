# DNSTT Protocol Prototype

This repository contains a compact prototype of a DNSTT-inspired tunnel protocol built in Go.

It demonstrates the core mechanics of a DNS tunnel design:

- segmenting payloads
- reassembly on the receiving side
- stream ID tracking
- frame checksums
- UDP datagram transport
- simple SYN/ACK/PING/PONG exchange

## Features

- custom binary frame format
- checksum validation
- fragmentation supported through sequence numbers
- stream multiplexing friendly design
- server/client demo over UDP

## Run

Start the server:

```bash
go run . -role=server -listen=127.0.0.1:5300
```

Start the client:

```bash
go run . -role=client -listen=127.0.0.1:5301 -peer=127.0.0.1:5300
```

## Notes

This is a protocol prototype, not a production-ready DNSTT implementation. It is intended as a foundation for:

- adaptive MTU sizing
- DNS TXT encoding
- selective ACKs
- retransmission logic
- compression / encryption
- real-world tunnel behavior

## Design Summary

The protocol uses a simple frame with:

- magic bytes (`DSNT`)
- frame type
- stream ID
- sequence number
- fragment index
- total fragments
- payload length
- checksum

This makes it possible to split large payloads into segments and reliably reassemble them on the other side.
