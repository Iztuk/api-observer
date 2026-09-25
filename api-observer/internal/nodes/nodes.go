// Package nodes handles node operations for
package nodes

import (
	"fmt"
	"sync"

	queryv1 "api-observer/proto/query/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	Name   string
	Addr   string
	Conn   *grpc.ClientConn
	Client queryv1.LogServiceClient
}

type NodeManager struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

func NewNodeManager() *NodeManager {
	return &NodeManager{
		nodes: make(map[string]*Node),
	}
}

func (m *NodeManager) Add(
	name string,
	addr string,
) error {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create client for node %q: %w",
			name,
			err,
		)
	}

	node := &Node{
		Name: name,
		Addr: addr,
		Conn: conn, Client: queryv1.NewLogServiceClient(conn),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.nodes[name]; ok {
		existing.Conn.Close()
	}

	m.nodes[name] = node

	return nil
}

func (m *NodeManager) Get(
	name string,
) (*Node, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, ok := m.nodes[name]

	return node, ok
}

func (m *NodeManager) List() []*Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make(
		[]*Node,
		0,
		len(m.nodes),
	)

	for _, node := range m.nodes {
		nodes = append(
			nodes,
			node,
		)
	}

	return nodes
}

func (m *NodeManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error

	for name, node := range m.nodes {
		if err := node.Conn.Close(); err != nil &&
			firstErr == nil {
			firstErr = fmt.Errorf(
				"failed to close node %q: %w",
				name,
				err,
			)
		}
	}

	return firstErr
}

func (m *NodeManager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[name]
	if !ok {
		return fmt.Errorf("node %q not found", name)
	}

	delete(m.nodes, name)

	if err := node.Conn.Close(); err != nil {
		return fmt.Errorf(
			"failed to close node %q: %w",
			name,
			err,
		)
	}

	return nil
}
