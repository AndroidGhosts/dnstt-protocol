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

func RandomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rnd.Intn(len(letters))]
    }
    return string(b)
}

type DNSTransport struct {
    Conn *net.UDPConn
}

func NewDNSTransport(listenAddr string) (*DNSTransport, error) {
    udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
    if err != nil {
        return nil, err
    }
    conn, err := net.ListenUDP("udp", udpAddr)
    if err != nil {
        return nil, err
    }
    return &DNSTransport{Conn: conn}, nil
}

func (d *DNSTransport) SendTo(targetAddr string, payload []byte) error {
    dst, err := net.ResolveUDPAddr("udp", targetAddr)
    if err != nil {
        return err
    }
    _, err = d.Conn.WriteToUDP(payload, dst)
    return err
}

func (d *DNSTransport) ReadFrom() ([]byte, string, error) {
    buf := make([]byte, 65535)
    n, remote, err := d.Conn.ReadFromUDP(buf)
    if err != nil {
        return nil, "", err
    }
    return buf[:n], remote.String(), nil
}

func (d *DNSTransport) Close() error {
    return d.Conn.Close()
}

func ToDNSLabel(s string) string {
    // Simple DNS-safe formatting for demo purposes.
    return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}

func ExampleDNSRecord(domain string, payload []byte) string {
    encoded := EncodeDNSPayload(payload)
    return fmt.Sprintf("%s.%s", ToDNSLabel(encoded[:min(len(encoded), 30)]), domain)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
