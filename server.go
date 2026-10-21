package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

func EncodeDNSPayload(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeDNSPayload(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty payload")
	}
	return base64.StdEncoding.DecodeString(s)
}

func BuildDNSQuestion(prefix, domain string) string {
	return prefix + "." + domain
}

func RandomDNSLabel(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}

type UDPTransport struct {
	Conn *net.UDPConn
}

func NewUDPTransport(listenAddr string) (*UDPTransport, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &UDPTransport{Conn: conn}, nil
}

func (u *UDPTransport) SendTo(peer string, payload []byte) error {
	dst, err := net.ResolveUDPAddr("udp", peer)
	if err != nil {
		return err
	}
	_, err = u.Conn.WriteToUDP(payload, dst)
	return err
}

func (u *UDPTransport) ReadFrom() ([]byte, string, error) {
	buf := make([]byte, 65535)
	n, remote, err := u.Conn.ReadFromUDP(buf)
	if err != nil {
		return nil, "", err
	}
	return buf[:n], remote.String(), nil
}

func (u *UDPTransport) Close() error {
	return u.Conn.Close()
}

func ExampleDNSRecord(domain string, payload []byte) string {
	encoded := EncodeDNSPayload(payload)
	safe := strings.ToLower(strings.ReplaceAll(encoded, "=", ""))
	if len(safe) > 30 {
		safe = safe[:30]
	}
	return fmt.Sprintf("%s.%s", safe, domain)
}
