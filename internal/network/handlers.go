package network

import (
	"encoding/binary"
	"io"
	"log"
	"net"

	"server/internal/game"
	"server/internal/utils"
)

func HandleTCPConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("New TCP connection from %s", conn.RemoteAddr())

	var client *game.Client

	defer func() {
		if client != nil {
			game.CleanupClient(client)
		}
		log.Printf("TCP connection from %s closed", conn.RemoteAddr())
	}()

	for {
		var msgLen uint32
		if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
			if err != io.EOF {
				log.Printf("Error reading message length from %s: %v", conn.RemoteAddr(), err)
			}
			return
		}

		if msgLen == 0 {
			continue
		}

		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, msgData); err != nil {
			log.Printf("Error reading message data from %s: %v", conn.RemoteAddr(), err)
			return
		}

		packetType := game.PacketType(binary.BigEndian.Uint32(msgData[:4]))
		payload := msgData[4:]

		switch packetType {
		case game.PacketConnect:
			if client != nil {
				log.Printf("Client %s sent duplicate connect message. Ignoring.", client.Oid)
				continue
			}

			game.Mutex.Lock()
			oid := utils.GenerateOID()
			client = &game.Client{
				Oid:     oid,
				TcpConn: conn,
			}
			game.Clients[oid] = client
			game.Mutex.Unlock()

			log.Printf("Client connected with OID: %s", oid)

			resPayload := []byte(oid)
			msg := make([]byte, 8+len(resPayload))
			binary.BigEndian.PutUint32(msg[0:4], uint32(game.PacketConnect))
			binary.BigEndian.PutUint32(msg[4:8], uint32(len(resPayload)))
			copy(msg[8:], resPayload)

			fullMsg := make([]byte, 4+len(msg))
			binary.BigEndian.PutUint32(fullMsg[0:4], uint32(len(msg)))
			copy(fullMsg[4:], msg)

			if _, err := conn.Write(fullMsg); err != nil {
				log.Printf("Failed to send connect response to %s: %v", conn.RemoteAddr(), err)
				return
			}

		case game.PacketHost:
			if client == nil {
				log.Println("Received host request from unauthenticated client.")
				return
			}

			game.Mutex.Lock()
			room := &game.Room{
				Host:    client,
				Clients: make(map[string]*game.Client),
				NextNID: 1,
			}
			client.Room = room
			client.NumericID = room.NextNID
			room.NextNID++
			room.Clients[client.Oid] = client

			game.Rooms[client.Oid] = room
			game.Mutex.Unlock()

			log.Printf("Client %s is hosting a new room.", client.Oid)
			game.BroadcastPeerList(room)

		case game.PacketJoin:
			if client == nil {
				log.Println("Received join request from unauthenticated client.")
				return
			}

			offset := 0
			oidLen := int(binary.BigEndian.Uint32(payload[offset : offset+4]))
			offset += 4
			offset += oidLen

			hostOidLen := int(binary.BigEndian.Uint32(payload[offset : offset+4]))
			offset += 4
			hostOid := string(payload[offset : offset+hostOidLen])

			game.Mutex.RLock()
			room, ok := game.Rooms[hostOid]
			game.Mutex.RUnlock()

			if !ok {
				log.Printf("Client %s tried to join non-existent room hosted by %s", client.Oid, hostOid)
				return
			}

			room.Lock()
			client.Room = room
			client.NumericID = room.NextNID
			room.NextNID++
			room.Clients[client.Oid] = client
			room.Unlock()

			log.Printf("Client %s (NID %d) joined room hosted by %s", client.Oid, client.NumericID, hostOid)
			game.BroadcastPeerList(room)

		case game.PacketLeaveRoom:
			if client == nil || client.Room == nil {
				continue
			}

			log.Printf("Client %s leaving room", client.Oid)
			room := client.Room

			msg := make([]byte, 4)
			binary.BigEndian.PutUint32(msg, uint32(game.PacketLeaveRoom))
			fullMsg := make([]byte, 8)
			binary.BigEndian.PutUint32(fullMsg[0:4], uint32(len(msg)))
			copy(fullMsg[4:], msg)
			conn.Write(fullMsg)

			game.CleanupClientInRoom(client)
			game.BroadcastPeerList(room)
			return
		}
	}
}

func HandleUDPPacket(conn *net.UDPConn, addr *net.UDPAddr, packet []byte) {
	if len(packet) < 8 {
		return
	}

	offset := 0
	senderOidLen := int(binary.BigEndian.Uint32(packet[offset : offset+4]))
	offset += 4
	if len(packet) < offset+senderOidLen+4 {
		return
	}
	senderOid := string(packet[offset : offset+senderOidLen])
	offset += senderOidLen

	targetOidLen := int(binary.BigEndian.Uint32(packet[offset : offset+4]))
	offset += 4
	if len(packet) < offset+targetOidLen {
		return
	}
	targetOid := string(packet[offset : offset+targetOidLen])
	offset += targetOidLen

	gameData := packet[offset:]

	game.Mutex.RLock()
	sender, ok := game.Clients[senderOid]
	game.Mutex.RUnlock()

	if !ok {
		return
	}

	if sender.UdpAddr == nil {
		sender.UdpAddr = addr
		log.Printf("Associated UDP address %s with client %s", addr.String(), sender.Oid)
	}

	if targetOid == "SERVER" {
		if string(gameData) == "UDP_CONNECT" {
			log.Printf("Received UDP_CONNECT from %s", senderOid)
			resPayload := []byte("UDP_CONNECT_RES")
			senderOidBytes := []byte("SERVER")
			targetOidBytes := []byte(sender.Oid)

			resPacket := make([]byte, 4+len(senderOidBytes)+4+len(targetOidBytes)+len(resPayload))
			resOffset := 0
			binary.BigEndian.PutUint32(resPacket[resOffset:resOffset+4], uint32(len(senderOidBytes)))
			resOffset += 4
			copy(resPacket[resOffset:resOffset+len(senderOidBytes)], senderOidBytes)
			resOffset += len(senderOidBytes)
			binary.BigEndian.PutUint32(resPacket[resOffset:resOffset+4], uint32(len(targetOidBytes)))
			resOffset += 4
			copy(resPacket[resOffset:resOffset+len(targetOidBytes)], targetOidBytes)
			resOffset += len(targetOidBytes)
			copy(resPacket[resOffset:], resPayload)

			conn.WriteToUDP(resPacket, sender.UdpAddr)
		}
		return
	}

	if sender.Room == nil {
		return
	}

	sender.Room.RLock()
	defer sender.Room.RUnlock()

	originalSenderOidBytes := []byte(sender.Oid)

	relayPacket := func(recipient *game.Client) {
		if recipient.UdpAddr == nil {
			return
		}
		targetOidBytes := []byte(recipient.Oid)
		newPacket := make([]byte, 4+len(originalSenderOidBytes)+4+len(targetOidBytes)+len(gameData))

		pktOffset := 0
		binary.BigEndian.PutUint32(newPacket[pktOffset:pktOffset+4], uint32(len(originalSenderOidBytes)))
		pktOffset += 4
		copy(newPacket[pktOffset:pktOffset+len(originalSenderOidBytes)], originalSenderOidBytes)
		pktOffset += len(originalSenderOidBytes)
		binary.BigEndian.PutUint32(newPacket[pktOffset:pktOffset+4], uint32(len(targetOidBytes)))
		pktOffset += 4
		copy(newPacket[pktOffset:pktOffset+len(targetOidBytes)], targetOidBytes)
		pktOffset += len(targetOidBytes)
		copy(newPacket[pktOffset:], gameData)

		conn.WriteToUDP(newPacket, recipient.UdpAddr)
	}

	if targetOid == "0" {
		for _, client := range sender.Room.Clients {
			if client.Oid != sender.Oid {
				relayPacket(client)
			}
		}
	} else {
		if recipient, ok := sender.Room.Clients[targetOid]; ok {
			relayPacket(recipient)
		}
	}
}
