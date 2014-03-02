package elect

import (
	//	"encoding/json"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"testing"
	"time"
	//	"encoding/gob"
	"io/ioutil"
)

// TestState checks whether server correctly
// reads persistent state from the disk
func TestState(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 1500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable"}

	// delete stored state to avoid unnecessary effect on following test cases
	deleteState(raftConf.StableStoreDirectoryPath)

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]Raft, serverCount+1)

	// create persistent state for one of the servers
	// state is created in such a way that one of the
	// servers will be way ahead of the others in 
	// logical time and hence it should eventually
	// be elected as reader
	pStateBytes, err := PersistentStateToBytes(&PersistentState{Term: 500, VotedFor: 1})


	if err != nil {
		//TODO: Add the state that was encoded. This might help in case of an error
		t.Errorf("Cannot encode PersistentState")
	}
	err = ioutil.WriteFile(raftConf.StableStoreDirectoryPath + "/1", pStateBytes, UserReadWriteMode)
	if err != nil {
		t.Errorf("Could not create state to storage on file " + raftConf.StableStoreDirectoryPath)
	}

	for i := 1; i <= serverCount; i += 1 {
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 5400+i, RaftToClusterConf(raftConf))
		if err != nil {
			t.Errorf("Error in creating cluster server. " + err.Error())
			return
		}
		s, err := NewWithConfig(clusterServer, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
			return
		}
		raftServers[i] = s
	}

	// there should be a leader after sufficiently long duration
	count := 0
	time.Sleep(10 * time.Second)
	leader := 0
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
			leader = i
			count++
		}
	}

	if count > 1 || leader != 1 {
		t.Errorf("Persistent state read was not utilized")
	}
}
