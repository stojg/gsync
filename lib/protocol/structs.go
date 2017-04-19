package protocol

import (
	"time"
)

type MessageType int

const (
	ClockSync MessageType = iota
	FileCreate
	FileChange
	FileDelete
	OK
	Done
)

type Message struct {
	CurrentTime time.Time
	Type        MessageType
	Content     string
}
