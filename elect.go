package elect

import (
	"github.com/pkhadilkar/cluster"
	"time"
)

// startElection starts raft election process upon
// receiving timeout. Untill a leader has been elected
// this routine continues process
func (s *raftServer) startElection() {
	// increment term and set state to candidate
	s.incrTerm()
	s.setState(CANDIDATE)
	// continue while state is candidate
	for s.getState() == CANDIDATE {
		// send RequestVote to all peers
		e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg : RequestVote{Term : s.Term(), CandidateId : s.server.Pid()}}
		s.server.Outbox() <- e
		// start a time for candidate time out
		
//		candidateTimeout := time.Duration(150 + s.rng.Intn(150)) // random timeout used by Raft authors
//		rTimer := time.NewTimer(candidateTimeout * time.Millisecond)
	}
	
}

// handleInbox examines messages received on inbox
// It forwards the messages to appropriate channels
func (s *raftServer) handleInbox() {
	for {
		e := <- s.server.Inbox()
		msg := e.Msg
		if voteMessage, ok := msg.(RequestVote); ok {
			s.elect <- &voteMessage
		} else if logMessage, ok := msg.(AppendEntry); ok {
			s.logEntry <- &logMessage
		}
	}
}


// revertToFollower changes state to follower and 
// updates server internal state
func (s *raftServer) revertToFollower(){
	
}

// respondToVote accepts RequestVote message and decides
// whether to send vote to the candidate
func (s *raftServer) respondToVoteRequest(msg *RequestVote) {

}

// serve is the main goroutine for Raft server
// This routine will maintain current server state and
// initiate election appropriately. This will serve as a
// "main" routine for a RaftServer
func (s *raftServer) serve() {
	s.handleInbox()
	for {
		select {
		case <- s.eTimeout.C: 
			// received timeout on election timer
			go s.startElection()
			
		case <- s.logEntry: 
			// received AppendEntries RPC message
			// currently we ignore the message and
			// treat every message as a heart beat
			go s.revertToFollower()

		case msg := <- s.elect:
			go s.respondToVoteRequest(msg)

		default : time.Sleep(1 * time.Millisecond) // sleep to avoid busy looping
		}
	}
}
