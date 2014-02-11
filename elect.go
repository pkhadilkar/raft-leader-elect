package elect

import (
	"github.com/pkhadilkar/cluster"
	"time"
)

// replyTo replies to sender of envelope
// msg is the reply content. Pid in the
// is used to identify the sender
func (s *raftServer) replyTo(e *cluster.Envelope, msg interface{}) {
	reply := &cluster.Envelope{Pid: e.Pid, Msg: msg}
	s.server.Outbox() <- reply
}

// serve is the main goroutine for Raft server
// This will serve as a "main" routine for a RaftServer.
// RaftServer's FOLLOWER and LEADER states are handled
// in this server routine.
func (s *raftServer) serve() {
	for {
		select {
		case e := <-s.server.Inbox():
			// received a message on server's inbox
			msg := e.Msg
			if ae, ok := msg.(AppendEntry); ok { // AppendEntry
				acc := false
				if ae.Term >= s.currentTerm {
					s.currentTerm = ae.Term
					acc = true
					if s.isLeader() {
						s.hbTimeout.Stop()
						s.setState(FOLLOWER)
					}
				}
				s.replyTo(e, &EntryReply{Term: s.currentTerm, Success: acc}) // can be asynchronous
			} else if rv, ok := msg.(RequestVote); ok { // RequestVote
				acc := false
				if (s.votedFor == e.Pid || s.votedFor == NotVoted) && rv.Term >= s.currentTerm || rv.Term > s.currentTerm {
					s.currentTerm = rv.Term
					s.votedFor = e.Pid
					// TODO: Force this on stable storage
					acc = true
				}
				s.replyTo(e, &GrantVote{Term: s.currentTerm, VoteGranted: acc})
			}

		case <-s.eTimeout.C:
			// received timeout on election timer
			s.startElection()
			if s.isLeader() {
				s.hbTimeout.Reset(s.hbDuration)
			}
		case <- s.hbTimeout.C :
			s.sendHeartBeat()
			s.hbTimeout.Reset(s.hbDuration)
		default:
			time.Sleep(1 * time.Millisecond) // sleep to avoid busy looping
		}
	}
}

// startElection handles election component of the
// raft server. Server stays in this function till
// it is in candidate state
func (s *raftServer) startElection() {
	s.setState(CANDIDATE)
	peers := s.server.Peers()
	votes := make(map[int]bool) // map to store received votes
	for s.getState() == CANDIDATE {
		s.incrTerm()                                                 // increment term for current
		candidateTimeout := time.Duration(150 + s.rng.Intn(150))     // random timeout used by Raft authors
		s.eTimeout.Reset(candidateTimeout * time.Millisecond) // start re-election timer
	innerLoop:
		for {
			select {
			case e := <-s.server.Inbox():
				// received a message on server's inbox
				msg := e.Msg
				if ae, ok := msg.(AppendEntry); ok { // AppendEntry
					acc := false
					if ae.Term >= s.currentTerm { // AppendEntry with same or larger term
						s.currentTerm = ae.Term
						s.setState(FOLLOWER)
						acc = true
					}
					s.replyTo(e, &EntryReply{Term: s.currentTerm, Success: acc}) // can be asynchronous
					if acc {
						break innerLoop
					}
				} else if rv, ok := msg.(RequestVote); ok { // RequestVote
					acc := false
					if rv.Term > s.currentTerm {
						s.currentTerm = rv.Term
						s.votedFor = e.Pid
						// TODO: Force this on stable storage
						s.setState(FOLLOWER)
						acc = true
					}
					s.replyTo(e, &GrantVote{Term: s.currentTerm, VoteGranted: acc})
					if acc {
						break innerLoop
					}
				} else if grantV, ok := msg.(GrantVote); ok && grantV.VoteGranted {
					votes[e.Pid] = true
					if len(votes) == len(peers) / 2 + 1 { // received majority votes
						s.setState(LEADER)
						s.sendHeartBeat()
						break innerLoop
					}
				}

			case <-s.eTimeout.C:
				// received timeout on election timer
				break innerLoop
			default:
				time.Sleep(1 * time.Millisecond) // sleep to avoid busy looping
			}
		}
	}
}


// sendHeartBeat sends heartbeat messages to followers
func (s *raftServer)sendHeartBeat() {

}
