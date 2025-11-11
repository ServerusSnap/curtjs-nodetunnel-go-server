package main

import (
	"server/internal/api"
	"server/internal/network"
)

func main() {
	// позже я отключу HTTP API, когда мне удастся добавить новые вичи в TCP сервер
	go network.StartTCPServer()
	go api.Server()

	network.StartUDPServer()
}
