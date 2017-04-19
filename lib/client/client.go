package client

import (
	"bufio"
	"net"
	"time"
)

func New(remoteAddress string) *Client {
	c := &Client{
		remoteAddress: remoteAddress,
	}
	return c
}

type Client struct {
	remoteAddress string // we store the remote address so we can reconnect in case of network blips
	conn          net.Conn
	rw            *bufio.ReadWriter
}

func (c *Client) RemoteAddress() string {
	return c.remoteAddress
}

func (c *Client) Conn() net.Conn {
	if c.conn == nil {
		if err := c.dial(); err != nil {
			panic(err)
		}
	}
	return c.conn
}

func (c *Client) Write(b []byte) (int, error) {
	if c.conn == nil {
		if err := c.dial(); err != nil {
			return 0, err
		}
	}
	nn, err := c.rw.Write(b)
	if err != nil {
		return nn, err
	}
	return nn, c.rw.Flush()
}

func (c *Client) dial() error {
	conn, err := net.DialTimeout("tcp", c.remoteAddress, time.Second)

	if err != nil {
		return err
	}
	c.conn = conn
	c.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	conn := c.conn
	c.conn = nil
	return conn.Close()
}
