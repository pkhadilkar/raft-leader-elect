/*
PseudoCluster is a cluster service that provides cluster.Server API.
Internally it stores a reference to cluster.Server object. It
includes an additional capability to filter messages. For each
PseudoCluster object, we can define a list of Pids for input and
output. If a PseudoCluster receives a message to/from a server in
these lists, it ignores the messages. PsuedoCluster can be used
to simulate server crash and network partitions.
*/
package test

import (
	"github.com/pkhadilkar/cluster"
	"sync"
)

const BufferSize = 100

type PseudoCluster struct {
	inbox          chan *cluster.Envelope
	outbox         chan *cluster.Envelope
	cluster.Server              // real cluster.Server object
	inboxFilter    map[int]bool // ignore messages received from server pids in this list on inbox
	outboxFilter   map[int]bool // ignore messages received on outbox for server pids in this list
	sync.Mutex                  // global mutex to access map. Efficiency is not a concern here
}

func NewPseudoCluster(clusterServer cluster.Server) *PseudoCluster {
	s := PseudoCluster{Server: clusterServer}
	s.inbox = make(chan *cluster.Envelope, BufferSize)
	s.outbox = make(chan *cluster.Envelope, BufferSize)
	s.inboxFilter = make(map[int]bool)
	s.outboxFilter = make(map[int]bool)
	go s.handleInbox()
	go s.handleOutbox()
	return &s
}

// AddToInboxFilter adds a new pid to inbox filter
// list. This function does not check whether same
// Pid already exists in the filter list.
// Note : This method is not thread-safe. Please
// ensure that only one method calls this at a time
func (s *PseudoCluster) AddToInboxFilter(pid int) {
	s.Lock()
	s.inboxFilter[pid] = true
	s.Unlock()
}

// AddToOutboxFilter adds a new pid to outbox filter
// list. This function does not check whether same
// Pid already exists in the filter list.
// Note : This method is not thread-safe. Please
// ensure that only one method calls this at a time
func (s *PseudoCluster) AddToOutboxFilter(pid int) {
	s.Lock()
	s.outboxFilter[pid] = true
	s.Unlock()
}

func (s *PseudoCluster) Inbox() chan *cluster.Envelope {
	return s.inbox
}

func (s *PseudoCluster) Outbox() chan *cluster.Envelope {
	return s.outbox
}

func (s *PseudoCluster) handleInbox() {
	for {
		e := <-s.Server.Inbox()
		// if message is received from cluster.Server that
		//is not in the filter list then send the message
		// on PseudoServer's inbox
		s.Lock()
		if _, exists := s.inboxFilter[e.Pid]; !exists {
			s.inbox <- e
		}
		s.Unlock()
	}
}

func (s *PseudoCluster) handleOutbox() {
	for {
		e := <-s.outbox
		// if message is received from server that is not
		// in the filter list then send the message on
		// cluster.Server's outbox
		s.Lock()
		if _, exists := s.outboxFilter[e.Pid]; !exists {
			s.Server.Outbox() <- e
		}
		s.Unlock()
	}
}

func (s *PseudoCluster) RemoveFromInboxFilter(pid int) {
	s.Lock()
	delete(s.inboxFilter, pid)
	s.Unlock()
}

func (s *PseudoCluster) RemoveFromOutboxFilter(pid int) {
	s.Lock()
	delete(s.outboxFilter, pid)
	s.Unlock()
}
