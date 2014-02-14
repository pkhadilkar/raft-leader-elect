package elect

import (
	"encoding/gob"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
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

// raftServer is a concrete implementation of raft Interface
type raftServer struct {
	currentTerm  int            // current term of the server
	currentState int            // current state of the server
	eTimeout     *time.Timer    // timer for election timeout
	hbTimeout    *time.Timer    // timer to send periodic hearbeats
	duration     time.Duration  // duration for election timeout
	hbDuration   time.Duration  // duration to send leader heartbeats
	votedFor     int            // id of the server that received vote from this server in current term
	server       cluster.Server // cluster server that provides message send/ receive functionality
	sync.Mutex                  // mutex to access the state atomically
	log          *log.Logger    // logger for server to store log messages
	rng          *rand.Rand
}

// Term returns current term of a raft server
func (s *raftServer) Term() int {
	//	s.Lock()
	currentTerm := s.currentTerm
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

// voteFor maintains the server side state
// to ensure that the server persists vote
func (s *raftServer) voteFor(pid int) {
	s.votedFor = pid
	//TODO: Force this on stable storage
}

// state returns the current state of the server
// value returned is one of the FOLLOWER,
// CANDIDATE and LEADER
func (s *raftServer) state() int {
	return s.currentState
}

// setState atomically sets the state of the server to newState
// TODO: Add verification for newState
func (s *raftServer) setState(newState int) {
	//	s.Lock()
	s.currentState = newState
	//	s.Unlock()
}

// incrTerm atomically increments server's term
func (s *raftServer) incrTerm() {
	//	s.Lock()
	s.currentTerm++
	//	s.Unlock()
}

//TODO: Load current term from persistent storage
func New(pid int, ip string, port int, configFile string) (Raft, error) {
	raftConfig, err := ReadConfig(configFile)
	if err != nil {
		fmt.Println("Error in reading config file.")
		return nil, err
	}
	// NewWithConfig(pid int, ip string, port int, raftConfig *RaftConfig) (Raft, error) {
	return NewWithConfig(pid, ip, port, raftConfig)
}

// function getLog creates a log for a raftServer
func getLog(s *raftServer, logDirPath string) error {
	fmt.Println("Creating log at " + logDirPath + "/" + strconv.Itoa(s.server.Pid()) + ".log")
	f, err := os.Create(logDirPath + "/" + strconv.Itoa(s.server.Pid()) + ".log")
	if err != nil {
		fmt.Println("Cannot create log files")
		return err
	}
	s.log = log.New(f, "", log.LstdFlags)
	return err
}

func NewWithConfig(pid int, ip string, port int, raftConfig *RaftConfig) (Raft, error) {
	s := raftServer{currentState: FOLLOWER, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	clusterConf := RaftToClusterConf(raftConfig)
	// initialize raft server details
	fmt.Println("NewWithConfig: Creating clusterServer")
	clusterServer, err := cluster.NewWithConfig(pid, ip, port, clusterConf)
	fmt.Println("Created cluster server")
	if err != nil {
		fmt.Println("Error in creating new instance of cluster server")
		return nil, err
	}

	s.server = clusterServer
	s.duration = time.Duration(raftConfig.TimeoutInMillis) * time.Millisecond
	s.hbDuration = time.Duration(raftConfig.HbTimeoutInMillis) * time.Millisecond
	s.eTimeout = time.NewTimer(s.duration + time.Duration(s.rng.Intn(150))) // start timer
	s.hbTimeout = time.NewTimer(s.duration)
	s.hbTimeout.Stop()
	s.votedFor = NotVoted

	err = getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}
	registerMessageTypes()
	go s.serve()
	return Raft(&s), err
}

// registerMessageTypes registers Message types used by
// server to gob. This is required because messages are
// defined as interfaces in cluster.Envelope
func registerMessageTypes() {
	gob.Register(AppendEntry{})
	gob.Register(EntryReply{})
	gob.Register(GrantVote{})
	gob.Register(RequestVote{})
}
