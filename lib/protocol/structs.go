package protocol

import (
	"time"
	"sync"
)

type MessageType int

const (
	ClockSync MessageType = iota
	FileDelete
	FileCreate
	Done
)

type Message struct {
	CurrentTime time.Time
	Type MessageType
}


type Config struct {
	sync.Mutex
	clockDrift time.Time
}
