package elect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

// Config struct represents all config information
// required to start a server. It represents
// information in config file in structure
type RaftConfig struct {
	MemberRegSocket string // socket to connect to , to register a cluster server
	PeerSocket      string // socket to connect to , to get a list of cluster peers
	TimeoutInMillis int64  // timeout duration to start a new Raft election
	HbTimeoutInMillis int64 // timeout to sent periodic heartbeats
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
