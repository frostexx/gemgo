package network

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// MockConnection represents a persistent connection to the network.
type MockConnection struct {
	id   int
	url  string
	// In a real app, this would hold a websocket or HTTP client connection.
	// For example: `conn *websocket.Conn`
}

type ConnectionPool struct {
	connections chan *MockConnection
	url         string
	maxSize     int
	mutex       sync.Mutex
}

func NewConnectionPool(url string, maxSize int) (*ConnectionPool, error) {
	if maxSize <= 0 {
		return nil, errors.New("pool size must be positive")
	}

	pool := &ConnectionPool{
		connections: make(chan *MockConnection, maxSize),
		url:         url,
		maxSize:     maxSize,
	}

	// Pre-warm the connections
	for i := 0; i < maxSize; i++ {
		conn := &MockConnection{id: i, url: url}
		pool.connections <- conn
	}
	log.Printf("Connection pool initialized with %d connections.", maxSize)
	return pool, nil
}

func (p *ConnectionPool) Get() *MockConnection {
	// This will block until a connection is available
	conn := <-p.connections
	return conn
}

func (p *ConnectionPool) Release(conn *MockConnection) {
	select {
	case p.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, which shouldn't happen with proper use.
		// We can log this anomaly.
		log.Println("Warning: Connection pool overflow on release.")
	}
}

func (p *ConnectionPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.connections != nil {
		close(p.connections)
		p.connections = nil
	}
}