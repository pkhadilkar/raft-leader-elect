package elect

import (
	//	"encoding/json"
	"./test"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"testing"
	"time"
	//	"encoding/gob"
)

// TestLeaderSeparation tests that new leader gets elected when
// elected leader crashes. Leader crash is simulated using
// PseudoCluster service
func TestLeaderSeparation(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 1500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs"}

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]*raftServer, serverCount+1)
	pseudoClusters := make([]*test.PseudoCluster, serverCount+1)
	for i := 1; i <= serverCount; i += 1 {
		// fmt.Println("Creating server " + strconv.Itoa(i))
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 5100+i, RaftToClusterConf(raftConf))
		pseudoCluster := test.NewPseudoCluster(clusterServer)
		if err != nil {
			t.Errorf("Error in creating cluster server. " + err.Error())
			return
		}
		s, err := NewWithConfig(pseudoCluster, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
			return
		}
		raftServers[i] = s.(*raftServer)
		pseudoClusters[i] = pseudoCluster
	}

	//fmt.Println("Created servers")
	// wait for leader to be elected
	time.Sleep(4 * time.Second)
	count := 0
	oldLeader := 0
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
			oldLeader = raftServers[i].Pid()
			count++
		}
	}
	if count != 1 {
		t.Errorf("No leader was chosen in 1 minute")
	}

	// isolate Leader
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].Pid() != oldLeader {
			pseudoClusters[oldLeader].AddToInboxFilter(raftServers[i].Pid())
			pseudoClusters[oldLeader].AddToOutboxFilter(raftServers[i].Pid())
		}
	}
	// prevent broadcasts from leader too
	pseudoClusters[oldLeader].AddToOutboxFilter(cluster.BROADCAST)
	// wait for other servers to discover that leader
	// has crashed and to elect a new leader
	time.Sleep(10 * time.Second)

	count = 0
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() && i != oldLeader {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as new leader.")
			count++
		}
	}
	// new leader must be chosen
	if count != 1 {
		t.Errorf("No leader was chosen")
	}

	// re-enable link between old leader and other servers
	for i := 1; i <= serverCount; i += 1 {
		s := raftServers[i]
		if s.Pid() != oldLeader {
			pseudoClusters[oldLeader].RemoveFromInboxFilter(s.Pid())
			pseudoClusters[oldLeader].RemoveFromOutboxFilter(s.Pid())
		}
	}

	// wait for oldLeader to discover new leader
	time.Sleep(2 * time.Second)

	// state of the old leader must be FOLLOWER
	// since a new leader has been elected
	// with a greater term
	if raftServers[oldLeader].state() != FOLLOWER {
		t.Errorf("Old leader is still in leader state.")
	}
}
