package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/stojg/gsync/lib/msg"
	"github.com/stojg/gsync/lib/server"
)

func runServer(signal chan os.Signal, address string, testDirectory string, numFiles int) error {
	s, err := server.New(address, serverHandler, testDirectory, numFiles)
	if err != nil {
		return err
	}
	fmt.Printf("running as server at %s\n", s.LocalAddress())
	sig := <-signal
	fmt.Println("\nGot signal:", sig)
	return nil
}

func serverHandler(conn net.Conn, dir string, num int) {
	fmt.Printf("client connected %s\n", conn.RemoteAddr())
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	// setup the random file content generator
	byteMaker := &randByteMaker{
		rand.NewSource(time.Now().Unix()),
	}

	var message msg.Message

	defer func() {
		if err := msg.Send(enc, msg.Done, "", nil); err != nil {
			log.Printf("encode error on exit: %s\n", err)
		}
		fmt.Println("\ntest done, closing client connection")
		conn.Close()
	}()

	if err := msg.Send(enc, msg.ClockSync, "", nil); err != nil {
		log.Println("message encoding error:", err)
		return
	}

	fileContent := &bytes.Buffer{}

	filesToCreate := fileOrder(num)
	fmt.Printf("will create and destroy %d files\n", len(filesToCreate))

	for _, file := range filesToCreate {

		// populate the file content from the random byte generator
		fileContent.Reset() // just being double
		io.CopyN(fileContent, byteMaker, file.size)

		tempFileName, err := createFile(dir, fileContent)
		if err != nil {
			log.Printf("%s\n", err)
			return
		}
		fmt.Print(".")

		tempBaseName := filepath.Base(tempFileName)
		if err := msg.Send(enc, msg.FileCreate, tempBaseName, &file.size); err != nil {
			log.Printf("\nmessage encoding error: %v\n", err)
			return
		}

		if err = dec.Decode(&message); err != nil {
			log.Printf("\nrecieve decode error: %v\n", err)
			return
		}

		// random jitter sleep to let the OS and network relax a bit and maybe avoid any OS level caching?
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

		err = os.Remove(tempFileName)
		if err != nil {
			log.Printf("\nremove error: %v\n", err)
			return
		}
		fmt.Print("x")

		if err := msg.Send(enc, msg.FileDelete, tempBaseName, nil); err != nil {
			log.Printf("\nmessage encoding error: %v\n", err)
			return
		}

		if err = dec.Decode(&message); err != nil {
			log.Printf("\nrecieve decode error: %v\n", err)
			return
		}
		// random jitter sleep to let the OS and network relax a bit and maybe avoid any OS level caching?
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	}
}

func createFile(dir string, content *bytes.Buffer) (string, error) {
	f, err := ioutil.TempFile(dir, "testfile_")
	if err != nil {
		return "", err
	}
	io.Copy(f, content)
	f.Close()
	return f.Name(), nil
}
