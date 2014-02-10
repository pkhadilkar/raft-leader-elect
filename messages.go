// This file contains details about raft messages. 
// The message formats are defined for raft in general.
// Thus, they should really be in a separate raft common
// package. For now, they are included here for convenience

package elect

// RequestVote struct is used in Raft leader election
type RequestVote struct {
	Term int // candidate's term
	CandidateId int // pid of candidate
}

// AppendEntries struct is used in Raft for sending
// log messages and hear beats. For this component
// it only contains term and leaderid
type AppendEntry struct {
	Term int // leader's term
	leaderId int // pid of the leader
}
