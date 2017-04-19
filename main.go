package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/stojg/gsync/lib/client"
	"github.com/stojg/gsync/lib/protocol"
	"github.com/stojg/gsync/lib/server"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var testDirectory string

func main() {

	runAsServer := flag.Bool("server", false, "run this as a server")
	serverAddress := flag.String("address", ":5666", "server address with port, eg 192.168.0.1:4302")

	flag.Parse()
	var err error

	testDirectory, err = filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if *runAsServer {
		err = runServer(c, *serverAddress)
	} else {
		err = runClient(*serverAddress)
	}
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runServer(signal chan os.Signal, address string) error {
	s, err := server.New(address, serverHandler)
	if err != nil {
		return err
	}
	log.Printf("running as server at %s\n", s.LocalAddress())
	sig := <-signal
	log.Println("\nGot signal:", sig)
	return nil
}

func runClient(address string) error {
	log.Println("running as client")
	c := client.New(address)
	log.Printf("connecting to server %s\n", address)
	err := clientHandler(c.Conn())
	if err != nil {
		return err
	}
	return nil
}

func serverHandler(conn net.Conn) {
	log.Printf("client connected %s\n", conn.RemoteAddr())
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)
	var msg protocol.Message

	defer func() {
		if err := sendMsg(enc, protocol.Done, ""); err != nil {
			log.Printf("encode error on exit: %s\n", err)
		}
		log.Println("test done, closing connection")
		conn.Close()
	}()

	if err := sendMsg(enc, protocol.ClockSync, ""); err != nil {
		log.Println("message encoding error:", err)
		return
	}

	for i := 0; i < 30; i++ {

		tempFileName, err := createFile(testDirectory)
		if err != nil {
			log.Printf("%s\n", err)
			return
		}

		tempBaseName := filepath.Base(tempFileName)
		if err := sendMsg(enc, protocol.FileCreate, tempBaseName); err != nil {
			log.Println("message encoding error:", err)
			return
		}

		log.Println("waiting for client")
		if err = dec.Decode(&msg); err != nil {
			log.Printf("recieve decode error: %s\n", err)
			return
		}

		// sleep a bit here to create some jitter
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		log.Printf("removing file %s\n", tempFileName)
		err = os.Remove(tempFileName)
		if err != nil {
			log.Printf("%s\n", err)
			return
		}
		if err := sendMsg(enc, protocol.FileDelete, tempBaseName); err != nil {
			log.Println("message encoding error:", err)
			return
		}

		log.Println("waiting for client")
		if err = dec.Decode(&msg); err != nil {
			log.Printf("recieve decode error: %s\n", err)
			return
		}
	}
}

func createFile(dirname string) (string, error) {
	f, err := ioutil.TempFile(dirname, "testfile_")
	if err != nil {
		return "", err
	}
	defer f.Close()

	log.Printf("created file %s\n", f.Name())
	return f.Name(), nil
}

func clientHandler(conn net.Conn) error {
	defer conn.Close()

	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	var msg protocol.Message

	var clockDrift time.Duration

	for {
		err := dec.Decode(&msg)
		if err != nil {
			return err
		}
		switch msg.Type {
		case protocol.Done:
			log.Println("Server is done, disconnecting")
			return nil
		case protocol.ClockSync:
			clockDrift = time.Since(msg.CurrentTime)
			log.Printf("clocks syncronised, drift %s\n", clockDrift)
		case protocol.FileCreate:
			if err := checkFile(msg.Content, msg.CurrentTime.Add(-clockDrift)); err != nil {
				return err
			}
			if err := sendMsg(enc, protocol.OK, ""); err != nil {
				return err
			}
		case protocol.FileDelete:
			if err := checkFileRemoved(msg.Content, msg.CurrentTime.Add(-clockDrift)); err != nil {
				return err
			}
			if err := sendMsg(enc, protocol.OK, ""); err != nil {
				return err
			}
		default:
			log.Printf("%+v\n", msg)
		}
	}
	return nil
}

func checkFile(tempBaseName string, serverTime time.Time) error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("created timed out after 5 seconds. no %s\n", tempBaseName)
		case <-ticker.C:
			stats, err := ioutil.ReadDir(testDirectory)
			if err != nil {
				return fmt.Errorf("readdir: %s\n", err)
			}
			for _, stat := range stats {
				if filepath.Base(stat.Name()) == tempBaseName {
					log.Printf("created %s, '%s', size: %d\n", time.Since(serverTime), tempBaseName, stat.Size())
					return nil
				}
			}
		}
	}
	// should never get down here
	return nil
}

func checkFileRemoved(tempBaseName string, serverTime time.Time) error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()

Loop:
	for {
		select {
		case <-timeout:
			return fmt.Errorf("delete timed out after 5 seconds, %s\n", tempBaseName)
		case <-ticker.C:
			stats, err := ioutil.ReadDir(testDirectory)
			if err != nil {
				return fmt.Errorf("readdir: %s\n", err)
			}

			for _, stat := range stats {
				if filepath.Base(stat.Name()) == tempBaseName {
					continue Loop
				}
			}
			log.Printf("deleted %s '%s'\n", time.Since(serverTime), tempBaseName)
			return nil
		}
	}
	return nil
}

func sendMsg(enc *gob.Encoder, t protocol.MessageType, content string) error {
	return enc.Encode(protocol.Message{
		CurrentTime: time.Now(),
		Type:        t,
		Content:     content,
	})
}
