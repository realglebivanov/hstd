package state

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ConnState struct {
	conn         *websocket.Conn
	sendCh       chan any
	done         chan struct{}
	wg           sync.WaitGroup
	shutdownOnce sync.Once
}

const (
	readTimeout  = 60 * time.Second
	pongTimeout  = 40 * time.Second
	pingInterval = 30 * time.Second
	writeTimeout = 5 * time.Second
)

var ConnClosed = errors.New("connection closed")

func New(c *websocket.Conn) (*ConnState, error) {
	if err := c.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, err
	}
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	s := &ConnState{
		conn:   c,
		sendCh: make(chan any, 16),
		done:   make(chan struct{}),
	}
	s.wg.Go(s.sendLoop)
	return s, nil
}

func (s *ConnState) Send(v any) error {
	select {
	case <-s.done:
		return ConnClosed
	default:
	}

	select {
	case s.sendCh <- v:
		return nil
	default:
		slog.Warn("ws send buffer full, dropping message")
		return fmt.Errorf("ws send buffer full")
	}
}

func (s *ConnState) ReadData() ([]byte, error) {
	select {
	case <-s.done:
		return nil, ConnClosed
	default:
		_, data, err := s.conn.ReadMessage()
		return data, err
	}
}

func (s *ConnState) Close() {
	s.shutdown()
	s.wg.Wait()
}

func (s *ConnState) shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.done)
		s.conn.Close()
	})
}

func (s *ConnState) sendLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer s.shutdown()
	for {
		select {
		case <-s.done:
			return
		case msg := <-s.sendCh:
			if err := s.conn.WriteJSON(msg); err != nil {
				slog.Warn("ws write", "err", err)
				return
			}
		case <-ticker.C:
			to := time.Now().Add(writeTimeout)
			if err := s.conn.WriteControl(websocket.PingMessage, nil, to); err != nil {
				slog.Warn("ws ping", "err", err)
				return
			}
		}
	}
}
