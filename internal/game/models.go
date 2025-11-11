package game

import (
	"net"
	"sync"
)

type PacketType uint32

const (
	PacketConnect PacketType = iota
	PacketHost
	PacketJoin
	PacketPeerList
	PacketLeaveRoom
)

type Client struct {
	Oid       string
	TcpConn   net.Conn
	UdpAddr   *net.UDPAddr
	NumericID uint32
	Room      *Room
}

type Room struct {
	Host    *Client
	Clients map[string]*Client
	NextNID uint32
	sync.RWMutex
}
