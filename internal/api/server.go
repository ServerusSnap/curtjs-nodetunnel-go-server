package api

import (
	"encoding/json"
	"log"
	"net/http"

	"server/internal/game"
)

func GetRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var roomList = make(map[string]int)
	game.Mutex.RLock()
	for _, room := range game.Rooms {
		roomList[room.Host.Oid] = len(room.Clients)
	}
	game.Mutex.RUnlock()
	log.Println(roomList)

	json.NewEncoder(w).Encode(roomList)
}

func Server() {
	http.HandleFunc("/api/rooms", GetRooms)
	log.Println("API server listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
