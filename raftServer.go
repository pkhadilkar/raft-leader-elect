package elect

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"math/rand"
	"sync"
	"time"
	"log"
	"os"
	"strconv"
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
	currentTerm int            // current term of the server
	leader      bool           // indicates whether current server is leader
	state       int            // current state of the server
	eTimeout    *time.Timer    // timer for election timeout
	hbTimeout   *time.Timer    // timer to send periodic hearbeats
	duration    time.Duration  // duration for election timeout
	hbDuration  time.Duration  // duration to send leader heartbeats
	votedFor    int            // id of the server that received vote from this server in current term
	server      cluster.Server // cluster server that provides message send/ receive functionality
	sync.Mutex                 // mutex to access the state atomically
	log *log.Logger // logger for server to store log messages
	rng         *rand.Rand
}

// Term returns current term of a raft server
func (s *raftServer) Term() int {
	s.Lock()
	currentTerm := s.currentTerm
	s.Unlock()
	return currentTerm
}

// setLeader sets value of leader flag to
// new value
func (s *raftServer) setLeader(leader bool) {
	s.leader = leader
}

// IsLeader returns true if r is a leader and false otherwise
func (s *raftServer) isLeader() bool {
	return s.leader
}

// state atomically reads state of the server
func (s *raftServer) getState() int {
	var currentState int
	s.Lock()
	currentState = s.state
	s.Unlock()
	return currentState
}

// setState atomically sets the state of the server to newState
// TODO: Add verification for newState
func (s *raftServer) setState(newState int) {
	//	s.Lock()
	s.state = newState
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
	s := raftServer{state: FOLLOWER, leader: false, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	raftConfig, err := ReadConfig(configFile)
	if err != nil {
		fmt.Println("Error in reading config file.")
		return nil, err
	}
	clusterConf := &cluster.Config{MemberRegSocket: raftConfig.MemberRegSocket, PeerSocket: raftConfig.PeerSocket}
	clusterServer, err := cluster.NewWithConfig(pid, ip, port, clusterConf)
	if err != nil {
		fmt.Println("Error in creating new instance of cluster server")
		return nil, err
	}

	// initialize raft server details
	s.server = clusterServer
	s.duration = time.Duration(raftConfig.TimeoutInMillis) * time.Millisecond
	s.hbDuration = time.Duration(raftConfig.HbTimeoutInMillis) * time.Millisecond
	s.eTimeout = time.NewTimer(s.duration) // start timer
	s.hbTimeout = time.NewTimer(s.duration)
	s.hbTimeout.Stop()                   // immediately stop the timer since hbTimer should only timeout on leader

	err = getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}
		go s.serve()
	return Raft(&s), err
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
	s := raftServer{state: FOLLOWER, leader: false, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
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
	s.eTimeout = time.NewTimer(s.duration) // start timer
	s.hbTimeout = time.NewTimer(s.duration)
	s.hbTimeout.Stop()

	err = getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}


	go s.serve()
	return Raft(&s), err	
}
