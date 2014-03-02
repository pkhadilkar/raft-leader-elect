package elect

import (
	//	"encoding/json"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"testing"
	"time"
	//	"encoding/gob"
	"os"
)

// TestElect tests normal behavior that leader should
// be elected under normal condition and everyone
// should agree upon current leader
func TestElect(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 1500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable"}

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]Raft, serverCount+1)

	for i := 1; i <= serverCount; i += 1 {
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 5000+i, RaftToClusterConf(raftConf))
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
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
			count++
		}
	}
	if count != 1 {
		t.Errorf("No leader was chosen")
	}

	// delete stored state to avoid unnecessary effect on following test cases
	deleteState(raftConf.StableStoreDirectoryPath)
}

// deleteState deletes persistent state on
// the disk for each server
// parameters:
//    baseDir: Path to base directory 
//     which contains state of all 
//     the servers on the disk
func deleteState(baseDir string) {
	base, err := os.Open(baseDir)
	if err != nil {
		fmt.Println("Error in opening directory." )
		return
	}
	fis, err := base.Readdir(-1) // read information for all files in the directory
	for _, f := range fis {
		err = os.Remove(baseDir + "/" + f.Name())
		if err != nil {
			fmt.Println("Error in deleting the file")
			fmt.Println(err.Error())
			return
		}
	}
	return
}
