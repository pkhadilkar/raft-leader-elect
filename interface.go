package elect

// Raft object interface.
// Term returns current server term
// isLeader returns true on leader and false on followers
type Raft interface {
	Term() int
	isLeader() bool
}
