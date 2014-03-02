package elect

import (
	"./test"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"strconv"
	"testing"
	"time"
)

// TestPartition tests that in case of network partition
// the partition with majority servers elect leader
func TestPartition(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 1500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable"}

	// delete stored state to avoid unnecessary effect on following test cases
	deleteState(raftConf.StableStoreDirectoryPath)

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]*raftServer, serverCount+1)
	pseudoClusters := make([]*test.PseudoCluster, serverCount+1)
	for i := 1; i <= serverCount; i += 1 {
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 5200+i, RaftToClusterConf(raftConf))
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

	// wait for leader to be elected
	time.Sleep(4 * time.Second)
	count := 0
	oldLeader := 0
	for i := 1; i <= serverCount; i += 1 {
		if raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
			oldLeader = i
			count++
		}
	}
	if count != 1 {
		t.Errorf("No leader was chosen in 1 minute")
	}

	// isolate Leader and any one follower
	follower := 0
	for i := 1; i <= serverCount; i += 1 {
		if i != oldLeader {
			follower = i
			break
		}
	}
	fmt.Println("Server " + strconv.Itoa(follower) + " was chosen as follower in minority partition")
	for i := 1; i <= serverCount; i += 1 {
		pseudoClusters[oldLeader].AddToInboxFilter(raftServers[i].Pid())
		pseudoClusters[oldLeader].AddToOutboxFilter(raftServers[i].Pid())
		pseudoClusters[follower].AddToInboxFilter(raftServers[i].Pid())
		pseudoClusters[follower].AddToOutboxFilter(raftServers[i].Pid())
	}

	pseudoClusters[oldLeader].AddToOutboxFilter(cluster.BROADCAST)
	pseudoClusters[follower].AddToOutboxFilter(cluster.BROADCAST)

	// wait for other servers to discover that leader
	// has crashed and to elect a new leader
	time.Sleep(9 * time.Second)

	count = 0
	for i := 1; i <= serverCount; i += 1 {
		if i != oldLeader && i != follower && raftServers[i].isLeader() {
			fmt.Println("Server " + strconv.Itoa(i) + " was chosen as new leader in majority partition.")
			count++
		}
	}
	// new leader must be chosen
	if count != 1 {
		t.Errorf("No leader was chosen in majority partition")
	}
}
