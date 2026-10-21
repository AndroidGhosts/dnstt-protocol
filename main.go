package main

import (
    "flag"
    "log"
    "time"
)

func main() {
    role := flag.String("role", "server", "server or client")
    listenAddr := flag.String("listen", "127.0.0.1:5300", "UDP listen address for this node")
    peerAddr := flag.String("peer", "127.0.0.1:5301", "peer UDP address")
    flag.Parse()

    switch *role {
    case "server":
        srv := NewServer(*listenAddr)
        log.Printf("server listening on %s", *listenAddr)

        go func() {
            time.Sleep(600 * time.Millisecond)
            if err := srv.SendText(*peerAddr, "hello-from-server"); err != nil {
                log.Printf("server send failed: %v", err)
            }
        }()

        srv.Run()
    case "client":
        cli := NewClient(*listenAddr, *peerAddr)
        log.Printf("client listening on %s, peer=%s", *listenAddr, *peerAddr)

        if err := cli.Connect(); err != nil {
            log.Fatalf("client connect failed: %v", err)
        }

        go func() {
            time.Sleep(300 * time.Millisecond)
            if err := cli.SendText("hello-from-client"); err != nil {
                log.Printf("client send failed: %v", err)
            }
        }()

        cli.Run()
    default:
        log.Fatalf("unknown role: %s", *role)
    }
}
