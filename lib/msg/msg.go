package msg

import (
	"encoding/gob"
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

func New(content string, msgType MessageType) *Message {
	return &Message{
		CurrentTime: time.Now(),
		Type:        msgType,
		FileName:    content,
	}
}

type Message struct {
	CurrentTime time.Time
	Type        MessageType
	FileName    string
	Size        int64
}

func Send(enc *gob.Encoder, t MessageType, content string, size *int64) error {
	m := Message{
		CurrentTime: time.Now(),
		Type:        t,
		FileName:    content,
	}
	if size != nil {
		m.Size = *size
	}
	return enc.Encode(m)
}
