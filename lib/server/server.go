package server

import (
	"fmt"
	"net"
)

func New(ladress string, handler func(net.Conn, string, int), testDirectory string, numFiles int) (*Server, error) {
	s := &Server{
		handler: handler,
		dir:     testDirectory,
		num:     numFiles,
	}

	address, err := net.ResolveTCPAddr("tcp", ladress)
	if err != nil {
		return s, err
	}
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return s, fmt.Errorf("Error listening: %v", err)
	}
	s.listener = listener
	go s.listen()
	return s, nil
}

type Server struct {
	listener     *net.TCPListener
	dir          string
	num          int
	handler      func(conn net.Conn, dir string, num int)
	localAddress string
	quit         chan bool
}

func (s *Server) LocalAddress() string {
	return s.listener.Addr().String()
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go s.handler(conn, s.dir, s.num)

	}
}
