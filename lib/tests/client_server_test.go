package client_test

import (
	"bufio"
	"github.com/silverstripeltd/gsync/lib/client"
	"github.com/silverstripeltd/gsync/lib/server"
	"net"
	"testing"
	"encoding/gob"
	"log"
	"github.com/silverstripeltd/gsync/lib/protocol"
	"time"
)

func TestNew(t *testing.T) {

	msgToSend := []byte("HELO localhost")
	var receivedMsg []byte
	done := make(chan bool)

	connHandler := func(conn net.Conn) {
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		var err error
		receivedMsg, err = rw.ReadBytes('\n')
		if err != nil {
			t.Errorf("Could not read from connection: %v\n", err)
		}
		receivedMsg = receivedMsg[:len(receivedMsg)-1]
		t.Logf("Read %d bytes from server\n", len(receivedMsg))
		done <- true
		defer conn.Close()
	}

	s, err := server.New(":", connHandler)
	if err != nil {
		t.Errorf("Did not expect error during server.New(): %v\n", err)
		return
	}
	defer s.Close()

	c := client.New(s.LocalAddress())
	if c == nil {
		t.Errorf("Did not expect error during client.New(): %v\n", err)
		return
	}
	defer c.Close()

	if c.RemoteAddress() != s.LocalAddress() {
		t.Errorf("expected client.RemoteAddress connect to %s, but got %s\n", s.LocalAddress(), c.RemoteAddress())
	}

	msg := append(msgToSend, '\n')
	n, err := c.Write(msg)
	if err != nil {
		t.Errorf("Error during client.Write: %v\n", err)
		return
	}

	if n != len(msg) {
		t.Errorf("Expected to client.Write to send %d bytes, but sent %d bytes\n", len(msgToSend), n)
		return
	}

	<-done

	if string(receivedMsg) != string(msgToSend) {
		t.Errorf("Expected to recieve message '%s', but got '%s'", msgToSend, receivedMsg)
	}
}

func TestGob(t *testing.T) {


	done := make(chan bool)

	// server
	connHandler := func(conn net.Conn) {
		enc := gob.NewEncoder(conn)
		err := enc.Encode(protocol.Message{
			CurrentTime: time.Now(),
			Type: protocol.ClockSync,
		})
		if err != nil {
			log.Fatal("encode error:", err)
		}
		done <- true
		defer conn.Close()
	}

	s, err := server.New(":", connHandler)
	if err != nil {
		t.Errorf("Did not expect error during server.New(): %v\n", err)
		return
	}
	defer s.Close()

	c := client.New(s.LocalAddress())
	if c == nil {
		t.Errorf("Did not expect error during client.New(): %v\n", err)
		return
	}
	defer c.Close()

	if c.RemoteAddress() != s.LocalAddress() {
		t.Errorf("expected client.RemoteAddress connect to %s, but got %s\n", s.LocalAddress(), c.RemoteAddress())
	}

	dec := gob.NewDecoder(c.Conn())

	var msg protocol.Message

	err = dec.Decode(&msg)
	if err != nil {
		t.Errorf("Error during gob.Decode: %v\n", err)
		return
	}


	t.Logf("client %v", msg)


	<-done

}
