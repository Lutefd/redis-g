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
type Server struct {
	Config
	ln        net.Listener
	peers     map[*Peer]bool
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan []byte
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
		msgCh:     make(chan []byte),
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

func (s *Server) handleRawMessage(rawMsg []byte) error {
	cmd, err := parseCommand(string(rawMsg))
	if err != nil {
		return err
	}
	switch v := cmd.(type) {
	case SetCommand:
		slog.Info("somebody  wants to set a key into the hash table", "key", v.key, "value", v.val)
		return s.kv.Set(v.key, v.val)
	}
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgCh:
			if err := s.handleRawMessage(rawMsg); err != nil {
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
	//block bc of errs
	slog.Info("new peer connected", "remoteAddr", conn.RemoteAddr())
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
	}
	time.Sleep(time.Second)
	fmt.Println(server.kv.data)
	select {} //blocking so the program doesn't exit
}
