package elect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"github.com/pkhadilkar/cluster"
	"strconv"
)

// Config struct represents all config information
// required to start a server. It represents
// information in config file in structure
type RaftConfig struct {
	MemberRegSocket   string // socket to connect to , to register a cluster server
	PeerSocket        string // socket to connect to , to get a list of cluster peers
	TimeoutInMillis   int64  // timeout duration to start a new Raft election
	HbTimeoutInMillis int64  // timeout to sent periodic heartbeats
	LogDirectoryPath string // path to log directory
}

// ReadConfig reads configuration file information into Config object
// parameters:
// path : Path to config file
func ReadConfig(path string) (*RaftConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf RaftConfig
	err = json.Unmarshal(data, &conf)
	if err != nil {
		fmt.Println("Error", err.Error())
		return nil, errors.New("Incorrect format in config file.\n" + err.Error())
	}
	return &conf, err
}

func RaftToClusterConf(r *RaftConfig) *cluster.Config {
	return &cluster.Config{MemberRegSocket: r.MemberRegSocket, PeerSocket: r.PeerSocket}
}

// writeToLog writes a formatted message to log
// It specifically adds server details to log
func (s *raftServer) writeToLog(msg string) {
	s.log.Println(strconv.Itoa(s.server.Pid())+ ": #" + strconv.Itoa(s.Term()) + ": \n" + msg)
}


// RequestVoteToJson converts a RequestVote msg 
// into a TypedMessage json and returns string
// representation of TypedMessage object
func RequestVoteToJson(r *RequestVote) (string, error) {
	tm := &TypedMessage{Type: REQ_VOTE, Msg: r}
	b, err := json.Marshal(tm)
	if err != nil {
		return "", err
	}
	return string(b), err
}

// AppendEntryToJson converts an AppendEntry msg 
// into a TypedMessage json and returns string
// representation of TypedMessage object
func AppendEntryToJson(r *AppendEntry) (string, error) {
	tm := &TypedMessage{Type: APP_ENTRY, Msg: r}
	b, err := json.Marshal(tm)
	if err != nil {
		return "", err
	}
	return string(b), err
}


// GrantVoteToJson converts an GrantVote msg 
// into a TypedMessage json and returns string
// representation of TypedMessage object
func GrantVoteToJson(r *GrantVote) (string, error) {
	tm := &TypedMessage{Type: GRANT_VOTE, Msg: r}
	b, err := json.Marshal(tm)
	if err != nil {
		return "", err
	}
	return string(b), err
}


// GrantVoteToJson converts an EntryReply msg 
// into a TypedMessage json and returns string
// representation of TypedMessage object
func EntryReplyToJson(r *EntryReply) (string, error) {
	tm := &TypedMessage{Type: ENTRY_REPLY, Msg: r}
	b, err := json.Marshal(tm)
	if err != nil {
		return "", err
	}
	return string(b), err
}
