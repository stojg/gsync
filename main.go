package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {

	runAsServer := flag.Bool("server", false, "run this as a server")
	serverAddress := flag.String("address", ":5666", "server address with port, eg 192.168.0.1:4302")
	numFiles := flag.Int("num", 50, "number of files to create and destroy")

	flag.Parse()

	testDirectory, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Directory under test: %s\n", testDirectory)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if *runAsServer {
		err = runServer(c, *serverAddress, testDirectory, *numFiles)
	} else {
		err = runClient(*serverAddress, testDirectory)
	}
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
