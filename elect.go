package elect

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"time"
)

// replyTo replies to sender of envelope
// msg is the reply content. Pid in the
// is used to identify the sender
func (s *raftServer) replyTo(to int, msg string) {
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
			s.writeToLog("server: Received a message on Server's inbox." + strconv.Itoa(e.Pid))
			msg := e.Msg
			if ae, ok := msg.(AppendEntry); ok { // AppendEntry
				s.handleAppendEntry(e.Pid, &ae)
			} else if rv, ok := msg.(RequestVote); ok { // RequestVote
				s.handleRequestVote(e.Pid, &rv)
			}

		case <-s.eTimeout.C:
			// received timeout on election timer
			s.writeToLog("Starting Election")
			// TODO: Use glog
			s.startElection()
			s.writeToLog("Election completed")
			if s.isLeader() {
				s.hbTimeout.Reset(s.hbDuration)
			}
		case <-s.hbTimeout.C:
			s.writeToLog("Sending hearbeats")
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
	s.writeToLog("Number of peers: " + strconv.Itoa(len(peers)))
	votes := make(map[int]bool) // map to store received votes
	votes[s.server.Pid()] = true
	s.voteFor(s.server.Pid())
	for s.state() == CANDIDATE {
		s.incrTerm()                                             // increment term for current
		candidateTimeout := time.Duration(150 + s.rng.Intn(500)) // random timeout used by Raft authors
		err := s.sendRequestVote()
		if err != nil {
			// Error here implies JSON error
			panic("StartElection: Received error" + err.Error())
		}
		s.writeToLog("Sent RequestVote message " + strconv.Itoa(int(candidateTimeout)))
		s.eTimeout.Stop()
		s.eTimeout.Reset(candidateTimeout * time.Millisecond) // start re-election timer
		for {
			acc := false
			select {
			case e, ok := <-s.server.Inbox():
				// received a message on server's inbox
				s.writeToLog("Received a MESSAGE on server's Inbox !!!!" + strconv.Itoa(e.Pid))
				fmt.Println(ok)
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
	if (s.votedFor == from || s.votedFor == NotVoted) && rv.Term >= s.currentTerm || rv.Term > s.currentTerm {
		s.currentTerm = rv.Term
		s.voteFor(from)
		if s.state() != FOLLOWER {
			s.follower()
		}
		acc = true
		s.writeToLog("Changing state to follower")
	}
	data, err := GrantVoteToJson(&GrantVote{Term: s.currentTerm, VoteGranted: acc})
	if err != nil {
		panic("ERROR: " + err.Error())
	}
	s.replyTo(from, data)
	return acc
}

// handleAppendEntry handles AppendEntry messages received
// when server is in CANDIDATE state
func (s *raftServer) handleAppendEntry(from int, ae *AppendEntry) bool {
	acc := false
	s.writeToLog("Received appendEntry message from " + strconv.Itoa(from) + " with term #" + strconv.Itoa(ae.Term))
	if ae.Term >= s.currentTerm { // AppendEntry with same or larger term
		s.currentTerm = ae.Term
		s.setState(FOLLOWER)
		acc = true
		s.writeToLog("Changing state to follower")
	}
	data, err := EntryReplyToJson(&EntryReply{Term: s.currentTerm, Success: acc})
	if err != nil {
		panic("ERROR: " + err.Error())
	}
	s.replyTo(from, data)
	return acc
}

// sendHeartBeat sends heartbeat messages to followers
func (s *raftServer) sendHeartBeat() {
	data, err := AppendEntryToJson(&AppendEntry{Term: s.Term(), LeaderId: s.server.Pid()})
	if err != nil {
		panic("ERROR: " + err.Error())
	}
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: data}
	s.server.Outbox() <- e
}

func (s *raftServer) sendRequestVote() error {
	rvJson, err := RequestVoteToJson(&RequestVote{Term: s.Term(), CandidateId: s.server.Pid()})
	if err != nil {
		s.log.Println("ERROR: Could not parse RequestVote Object to JSON")
		return err
	}
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: rvJson}
	s.writeToLog("Sending message (Pid: " + strconv.Itoa(e.Pid) + ", CandidateId: " + strconv.Itoa(s.server.Pid()))
	s.server.Outbox() <- e
	return err
}
