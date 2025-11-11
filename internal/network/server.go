package network

import (
	"log"
	"net"
)

const (
	TCP_PORT = "9998"
	UDP_PORT = "9999"
)

func StartTCPServer() {
	listener, err := net.Listen("tcp", ":"+TCP_PORT)
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()
	log.Printf("TCP server listening on port %s", TCP_PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept TCP connection: %v", err)
			continue
		}
		go HandleTCPConnection(conn)
	}
}

func StartUDPServer() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+UDP_PORT)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	defer conn.Close()
	log.Printf("UDP server listening on port %s", UDP_PORT)

	buffer := make([]byte, 2048)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Failed to read from UDP: %v", err)
			continue
		}

		go HandleUDPPacket(conn, addr, buffer[:n])
	}
}
