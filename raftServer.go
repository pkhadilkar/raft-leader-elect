// Raft leader election launches a set of Raft processes and
// ensures that a leader is elected. State of the Raft 
// servers is stored on stable storage / disk.

package elect

import (
	"encoding/gob"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const bufferSize = 100

const NotVoted = -1

//TODO: Use separate locks for state and term

// constants for states of the server
const (
	FOLLOWER = iota
	LEADER
	CANDIDATE
)

// TODO: store pointer to raftConfig rather than storing all fields in raftServer
// raftServer is a concrete implementation of raft Interface
type raftServer struct {
	currentState int            // current state of the server
	eTimeout     *time.Timer    // timer for election timeout
	hbTimeout    *time.Timer    // timer to send periodic hearbeats
	duration     int64          // duration for election timeout
	hbDuration   int64          // duration to send leader heartbeats
	server       cluster.Server // cluster server that provides message send/ receive functionality
	log          *log.Logger    // logger for server to store log messages
	rng          *rand.Rand
	state        *PersistentState // server information that should be persisted
	config       *RaftConfig      // config information for raftServer
}

// Term returns current term of a raft server
func (s *raftServer) Term() int {
	//	s.Lock()
	currentTerm := s.state.Term
	//	s.Unlock()
	return currentTerm
}

// IsLeader returns true if r is a leader and false otherwise
func (s *raftServer) isLeader() bool {
	return s.currentState == LEADER
}

// follower changes current state of the server
// to follower
func (s *raftServer) follower() {
	s.hbTimeout.Stop()
	s.setState(FOLLOWER)
}

// SetTerm sets the current term of the server
// The changes are persisted on disk
func (s *raftServer) setTerm(term int) {
	s.state.Term = term
	s.persistState()
}

// voteFor maintains the server side state
// to ensure that the server persists vote
// Parameters:
//  pid  : pid of the server to whom vote is given
//  term : term for which the vote was granted
func (s *raftServer) voteFor(pid int, term int) {
	s.state.VotedFor = pid
	s.state.Term = term
	//TODO: Force this on stable storage
	s.persistState()
}

// persistState persists the state of the server on
// stable storage. This function panicks if it cannot
// write the state.
func (s *raftServer) persistState() {
	pStateBytes, err := PersistentStateToBytes(s.state)
	if err != nil {
		//TODO: Add the state that was encoded. This might help in case of an error
		panic("Cannot encode PersistentState")
	}
	err = ioutil.WriteFile(s.config.StableStoreDirectoryPath+"/"+strconv.Itoa(s.server.Pid()), pStateBytes, UserReadWriteMode)
	if err != nil {
		panic("Could not persist state to storage on file " + s.config.StableStoreDirectoryPath)
	}
}

// readPersistentState tries to read the persistent
// state from the stable storage. It does not panic if
// no record of state is found in stable storage as
// that represents a valid state when server is  started
// for the first time
func (s *raftServer) readPersistentState() {
	pStateRead, err := ReadPersistentState(s.config.StableStoreDirectoryPath)
	if err != nil {
		s.state = &PersistentState{VotedFor: NotVoted, Term: 0}
		return
	}
	s.state = pStateRead
}

// state returns the current state of the server
// value returned is one of the FOLLOWER,
// CANDIDATE and LEADER
func (s *raftServer) State() int {
	return s.currentState
}

// votedFor returns the pid of the server
// voted for by current server
func (s *raftServer) VotedFor() int {
	return s.state.VotedFor
}

// setState sets the state of the server to newState
// TODO: Add verification for newState
func (s *raftServer) setState(newState int) {
	//	s.Lock()
	s.currentState = newState
	//	s.Unlock()
}

func (s *raftServer) Pid() int {
	return s.server.Pid()
}

// incrTerm  increments server's term
func (s *raftServer) incrTerm() {
	//	s.Lock()
	s.state.Term++
	//	s.Unlock()
	// TODO: Is persistState necessary here ?
	s.persistState()
}

//TODO: Load current term from persistent storage
func New(clusterServer cluster.Server, configFile string) (Raft, error) {
	raftConfig, err := ReadConfig(configFile)
	if err != nil {
		fmt.Println("Error in reading config file.")
		return nil, err
	}
	// NewWithConfig(pid int, ip string, port int, raftConfig *RaftConfig) (Raft, error) {
	return NewWithConfig(clusterServer, raftConfig)
}

// function getLog creates a log for a raftServer
func getLog(s *raftServer, logDirPath string) error {
	fmt.Println("Creating log at " + logDirPath + "/" + strconv.Itoa(s.server.Pid()) + ".log")
	f, err := os.Create(logDirPath + "/" + strconv.Itoa(s.server.Pid()) + ".log")
	if err != nil {
		fmt.Println("Error: Cannot create log files")
		return err
	}
	s.log = log.New(f, "", log.LstdFlags)
	return err
}

// NewWithConfig creates a new raftServer.
// Parameters:
//  clusterServer : Server object of cluster API. cluster.Server provides message send
//                  receive along with other facilities such as finding peers
//  raftConfig    : Raft configuration object
func NewWithConfig(clusterServer cluster.Server, raftConfig *RaftConfig) (Raft, error) {
	s := raftServer{currentState: FOLLOWER, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	s.server = clusterServer
	s.config = raftConfig
	// read persistent state from the disk if server was being restarted as a
	// part of recovery then it would find the persistent state on the disk
	s.readPersistentState()
	s.duration = raftConfig.TimeoutInMillis
	s.hbDuration = raftConfig.HbTimeoutInMillis
	s.eTimeout = time.NewTimer(time.Duration(s.duration+s.rng.Int63n(150)) * time.Millisecond) // start timer
	s.hbTimeout = time.NewTimer(time.Duration(s.duration) * time.Millisecond)
	s.hbTimeout.Stop()
	s.state.VotedFor = NotVoted

	err := getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}
	
	go s.serve()
	return Raft(&s), err
}

// registerMessageTypes registers Message types used by
// server to gob. This is required because messages are
// defined as interfaces in cluster.Envelope
func init() {
	gob.Register(AppendEntry{})
	gob.Register(EntryReply{})
	gob.Register(GrantVote{})
	gob.Register(RequestVote{})
}
