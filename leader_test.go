package elect

import (
	//	"encoding/json"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"testing"
	"time"
	//	"encoding/gob"
)

// TestElect tests normal behavior that leader should
// be elected under normal condition and everyone
// should agree upon current leader
func TestElect(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 150, HbTimeoutInMillis: 50, LogDirectoryPath: "/tmp/logs"}

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]Raft, serverCount+1)

	for i := 1; i <= serverCount; i += 1 {
		fmt.Println("Creating server " + strconv.Itoa(i))
		s, err := NewWithConfig(i, "127.0.0.1", 5000+i, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
		}
		raftServers[i] = s
	}

	fmt.Println("Created servers")

	// there should be a leader after sufficiently long duration

	time.Sleep(1 * time.Minute)
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
			return
		}
	}
	t.Errorf("No leader was chosen in 1 minute")
}
