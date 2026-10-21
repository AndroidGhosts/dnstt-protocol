package main

import (
    "flag"
    "log"
)

func main() {
    role := flag.String("role", "server", "server or client")
    listenAddr := flag.String("listen", "127.0.0.1:5300", "UDP listen address for this node")
    peerAddr := flag.String("peer", "127.0.0.1:5301", "Peer UDP address")
    flag.Parse()

    switch *role {
    case "server":
        srv, err := NewServer(*listenAddr)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("server listening on %s", *listenAddr)
        srv.Run()
    case "client":
        cli, err := NewClient(*listenAddr, *peerAddr)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("client listening on %s, peer=%s", *listenAddr, *peerAddr)
        if err := cli.Connect(); err != nil {
            log.Fatal(err)
        }
        cli.Run()
    default:
        log.Fatalf("unknown role: %s", *role)
    }
}
