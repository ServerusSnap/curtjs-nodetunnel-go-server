package game

import (
	"encoding/binary"
	"log"
)

func BroadcastPeerList(room *Room) {
	room.RLock()
	defer room.RUnlock()

	if len(room.Clients) == 0 {
		return
	}

	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(len(room.Clients)))

	for _, c := range room.Clients {
		oidBytes := []byte(c.Oid)
		pData := make([]byte, 4+len(oidBytes)+4)
		binary.BigEndian.PutUint32(pData[0:4], uint32(len(oidBytes)))
		copy(pData[4:4+len(oidBytes)], oidBytes)
		binary.BigEndian.PutUint32(pData[4+len(oidBytes):], c.NumericID)
		payload = append(payload, pData...)
	}

	msg := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(msg[0:4], uint32(PacketPeerList))
	copy(msg[4:], payload)

	fullMsg := make([]byte, 4+len(msg))
	binary.BigEndian.PutUint32(fullMsg[0:4], uint32(len(msg)))
	copy(fullMsg[4:], msg)

	for _, c := range room.Clients {
		if c.TcpConn != nil {
			if _, err := c.TcpConn.Write(fullMsg); err != nil {
				log.Printf("Failed to broadcast peer list to %s: %v", c.Oid, err)
			}
		}
	}
	log.Printf("Broadcasted peer list to %d clients in room hosted by %s", len(room.Clients), room.Host.Oid)
}

func CleanupClientInRoom(client *Client) {
	if client == nil || client.Room == nil {
		return
	}

	room := client.Room
	room.Lock()
	defer room.Unlock()

	if _, ok := room.Clients[client.Oid]; !ok {
		return
	}

	isHost := room.Host == client
	delete(room.Clients, client.Oid)
	client.Room = nil
	log.Printf("Removed client %s from room", client.Oid)

	if isHost {
		log.Printf("Host %s disconnected, closing room", client.Oid)
		Mutex.Lock()
		delete(Rooms, client.Oid)
		Mutex.Unlock()

		for _, otherClient := range room.Clients {
			otherClient.Room = nil
		}
		room.Clients = make(map[string]*Client)
	}
}

func CleanupClient(client *Client) {
	if client == nil {
		return
	}

	CleanupClientInRoom(client)

	Mutex.Lock()
	delete(Clients, client.Oid)
	Mutex.Unlock()

	log.Printf("Finished cleanup for client %s", client.Oid)
}
