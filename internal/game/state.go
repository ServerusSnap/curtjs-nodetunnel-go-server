package game

import "sync"

var (
	Clients = make(map[string]*Client)
	Rooms   = make(map[string]*Room)
	Mutex   = &sync.RWMutex{}
)
