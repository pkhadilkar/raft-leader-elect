package elect

import (
	"github.com/pkhadilkar/cluster"
	"strconv"
	"time"
)

// RandomTimeoutRange indicates number of milliseconds
// randome timeout should vary from its base value
const RandomTimeoutRange = 150

// replyTo replies to sender of envelope
// msg is the reply content. Pid in the
// is used to identify the sender
func (s *raftServer) replyTo(to int, msg interface{}) {
	reply := &cluster.Envelope{Pid: to, Msg: msg}
	s.server.Outbox() <- reply
}

// serve is the main goroutine for Raft server
// This will serve as a "main" routine for a RaftServer.
// RaftServer's FOLLOWER and LEADER states are handled
// in this server routine.
func (s *raftServer) serve() {
	s.writeToLog("Started serve")
	for {
		select {
		case e := <-s.server.Inbox():
			// received a message on server's inbox
			msg := e.Msg
			if ae, ok := msg.(AppendEntry); ok { // AppendEntry
				acc := s.handleAppendEntry(e.Pid, &ae)
				if acc {
					candidateTimeout := time.Duration(s.duration + s.rng.Int63n(RandomTimeoutRange))
					// reset election timer if valid message received from server
					s.eTimeout.Reset(candidateTimeout * time.Millisecond)
				}
			} else if rv, ok := msg.(RequestVote); ok { // RequestVote
				s.handleRequestVote(e.Pid, &rv) // reset election timeout here too ? To avoid concurrent elections ?
			}

			// TODO handle EntryReply message

		case <-s.eTimeout.C:
			// received timeout on election timer
			s.writeToLog("Starting Election")
			// TODO: Use glog
			s.startElection()
			s.writeToLog("Election completed")
			if s.isLeader() {
				s.hbTimeout.Reset(time.Duration(s.hbDuration) * time.Millisecond)
				s.eTimeout.Stop() // leader should not time out for election
			}
		case <-s.hbTimeout.C:
			s.writeToLog("Sending hearbeats")
			s.sendHeartBeat()
			s.hbTimeout.Reset(time.Duration(s.hbDuration) * time.Millisecond)
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
	s.writeToLog("Number of peers: " + strconv.Itoa(len(peers)))
	votes := make(map[int]bool) // map to store received votes
	votes[s.server.Pid()] = true
	s.voteFor(s.server.Pid(), s.Term())
	for s.State() == CANDIDATE {
		s.incrTerm()                                                                     // increment term for current
		candidateTimeout := time.Duration(s.duration + s.rng.Int63n(RandomTimeoutRange)) // random timeout used by Raft authors
		s.sendRequestVote()
		s.writeToLog("Sent RequestVote message " + strconv.Itoa(int(candidateTimeout)))
		s.eTimeout.Reset(candidateTimeout * time.Millisecond) // start re-election timer
		for {
			acc := false
			select {
			case e, _ := <-s.server.Inbox():
				// received a message on server's inbox
				msg := e.Msg
				if ae, ok := msg.(AppendEntry); ok { // AppendEntry
					acc = s.handleAppendEntry(e.Pid, &ae)
				} else if rv, ok := msg.(RequestVote); ok { // RequestVote
					acc = s.handleRequestVote(e.Pid, &rv)

				} else if grantV, ok := msg.(GrantVote); ok && grantV.VoteGranted {
					votes[e.Pid] = true
					s.writeToLog("Received grantVote message from " + strconv.Itoa(e.Pid) + " with term #" + strconv.Itoa(grantV.Term))
					s.writeToLog("Votes received so far " + strconv.Itoa(len(votes)))
					if len(votes) == len(peers)/2+1 { // received majority votes
						s.setState(LEADER)
						s.sendHeartBeat()
						acc = true
					}
				}
			case <-s.eTimeout.C:
				// received timeout on election timer
				s.writeToLog("Received re-election timeout")
				acc = true
			default:
				time.Sleep(1 * time.Millisecond) // sleep to avoid busy looping
			}

			if acc {
				s.eTimeout.Reset(candidateTimeout * time.Millisecond) // start re-election timer
				break
			}
		}
	}
}

// handleRequestVote  handles RequestVote messages
// when server is in candidate state
func (s *raftServer) handleRequestVote(from int, rv *RequestVote) bool {
	acc := false
	// in currentTerm candidate votes for itself
	s.writeToLog("Received requestVote message from " + strconv.Itoa(from) + " with term #" + strconv.Itoa(rv.Term))
	if (s.VotedFor() == from || s.VotedFor() == NotVoted) && rv.Term >= s.Term() || rv.Term > s.Term() {
		s.setTerm(rv.Term)
		s.voteFor(from, s.Term())
		if s.State() != FOLLOWER {
			s.follower()
		}
		acc = true
		s.writeToLog("Granting vote to " + strconv.Itoa(from) + ".Changing state to follower")
	}
	s.replyTo(from, &GrantVote{Term: s.Term(), VoteGranted: acc})
	return acc
}

// handleAppendEntry handles AppendEntry messages received
// when server is in CANDIDATE state
func (s *raftServer) handleAppendEntry(from int, ae *AppendEntry) bool {
	acc := false
	s.writeToLog("Received appendEntry message from " + strconv.Itoa(from) + " with term #" + strconv.Itoa(ae.Term))
	if ae.Term >= s.Term() { // AppendEntry with same or larger term
		s.setTerm(ae.Term)
		s.setState(FOLLOWER)
		acc = true
	}
	s.replyTo(from, &EntryReply{Term: s.Term(), Success: acc})
	return acc
}

// sendHeartBeat sends heartbeat messages to followers
func (s *raftServer) sendHeartBeat() {
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: &AppendEntry{Term: s.Term(), LeaderId: s.server.Pid()}}
	s.server.Outbox() <- e
}

func (s *raftServer) sendRequestVote() {
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: &RequestVote{Term: s.Term(), CandidateId: s.server.Pid()}}
	s.writeToLog("Sending message (Pid: " + strconv.Itoa(e.Pid) + ", CandidateId: " + strconv.Itoa(s.server.Pid()))
	s.server.Outbox() <- e
}
