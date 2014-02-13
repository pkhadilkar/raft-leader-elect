// This file contains details about raft messages.
// The message formats are defined for raft in general.
// Thus, they should really be in a separate raft common
// package. For now, they are included here for convenience

package elect

// enum to indicate type of the message
const (
	REQ_VOTE = iota
	APP_ENTRY
	GRANT_VOTE
	ENTRY_REPLY
)


// RequestVote struct is used in Raft leader election
type RequestVote struct {
	Term        int // candidate's term
	CandidateId int // pid of candidate
}

// AppendEntries struct is used in Raft for sending
// log messages and hear beats. For this component
// it only contains term and leaderid
type AppendEntry struct {
	Term     int // leader's term
	LeaderId int // pid of the leader
}

type GrantVote struct {
	Term        int // currentTerm for candidate
	VoteGranted bool
}

// EntryReply is reply message for AppendEntry request
type EntryReply struct {
	Term    int  // replying server's updated current term
	Success bool // true if AppendEntry was accepted
}

// TypedMessage is a message with data and type of the 
// data. Messages sent between servers use these messages
// as this simpliefies decoding. The type is indicated by
// constants defined above
type TypedMessage struct {
	Type int
	Msg interface{}
}
