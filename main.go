package main

import (
	"flag"
	"fmt"
	"os"
	"net"
	"encoding/gob"
	"time"
	"log"
	"github.com/silverstripeltd/gsync/lib/protocol"
	"github.com/silverstripeltd/gsync/lib/server"
	"os/signal"
	"github.com/silverstripeltd/gsync/lib/client"
)

func main() {

	runAsServer := flag.Bool("server", false, "run this as a server")
	serverAddress := flag.String("address", ":5666", "server address with port, eg 192.168.0.1:4302")
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	var err error
	if *runAsServer {
		err = runServer(c, *serverAddress)
	} else {
		err = runClient(*serverAddress)
	}
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runServer(signal chan os.Signal, address string) error {
	s, err := server.New(address, serverHandler)
	if err != nil {
		return err
	}
	fmt.Printf("running as server at %s\n", s.LocalAddress())
	sig := <-signal
	fmt.Println("\nGot signal:", sig)
	return nil
}

func runClient(address string) error {
	fmt.Println("running as client")
	c := client.New(address)

	fmt.Printf("Connecting to %s\n", address)

	err := clientHandler(c.Conn())
	if err != nil {
		return err
	}

	return nil
}

func serverHandler(conn net.Conn) {
	fmt.Printf("Client connected %s\n", conn.RemoteAddr())
	enc := gob.NewEncoder(conn)

	defer func() {
		err := enc.Encode(protocol.Message{
			CurrentTime: time.Now(),
			Type: protocol.Done,
		})
		if err != nil {
			fmt.Println("encode error on exit: %s\n", err)
		}
		conn.Close()
	}()


	err := enc.Encode(protocol.Message{
		CurrentTime: time.Now(),
		Type: protocol.ClockSync,
	})
	if err != nil {
		log.Fatal("encode error:", err)
	}

	f, err := os.Create("test_file")
	if err {
		fmt.Println("%s\n", err)
		return
	}
}

func clientHandler(conn net.Conn) error {
	defer conn.Close()

	dec := gob.NewDecoder(conn)
	var msg protocol.Message


	var clockDrift time.Duration

	for {
		err := dec.Decode(&msg)
		if err != nil {
			return err
		}
		switch msg.Type {
		case protocol.Done:
			fmt.Println("Server is done, disconnecting")
			return nil
		case protocol.ClockSync:
			clockDrift = time.Since(msg.CurrentTime)
			fmt.Printf("Clocks syncronised, drift %s\n", clockDrift)
		}
	}

	return nil

}
