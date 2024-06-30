package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/lutefd/redis-g/client"
)

const defaultListenAddr = ":5001"

type Config struct {
	ListenAddr string
}
type Message struct {
	data []byte
	peer *Peer
}
type Server struct {
	Config
	ln        net.Listener
	peers     map[*Peer]bool
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message
	kv        *KV
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan Message),
		kv:        NewKV(),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()

	slog.Info("server running", "listenAddr", s.ListenAddr)
	return s.acceptLoop()
}

func (s *Server) handleMessage(msg Message) error {
	cmd, err := parseCommand(string(msg.data))
	if err != nil {
		return err
	}
	switch v := cmd.(type) {
	case SetCommand:
		return s.kv.Set(v.key, v.val)
	case GetCommand:
		val, ok := s.kv.Get(v.key)
		if !ok {
			return fmt.Errorf("key not found")
		}
		_, err := msg.peer.Send(val)
		if err != nil {
			return fmt.Errorf("peer send error, err: %s ", err)
		}
	}

	return nil
}

func (s *Server) loop() {
	for {
		select {
		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				slog.Error("raw message decoding error", "err", err)

			}
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		case <-s.quitCh:
			return
		}
	}
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("Accept error", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", "err", err, "remoteAddr", conn.RemoteAddr())
	}
}

func main() {
	server := NewServer(Config{})
	go func() {
		log.Fatal(server.Start())
	}()
	time.Sleep(time.Second)
	c := client.New("localhost:5001")
	for i := 0; i < 10; i++ {
		if err := c.Set(context.TODO(), fmt.Sprintf("foo %d", i), fmt.Sprintf("bar %d", i)); err != nil {
			log.Fatal(err)
		} // this adding peers for no reason at all, will fix it later
		val, err := c.Get(context.TODO(), fmt.Sprintf("foo %d", i))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("got this back =>", val)
	}
}
