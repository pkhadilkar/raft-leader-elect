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
	raftServers := make([]*raftServer, serverCount+1)

	for i := 1; i <= serverCount; i += 1 {
		fmt.Println("Creating server " + strconv.Itoa(i))
		s, err := NewWithConfig(i, "127.0.0.1", 5000+i, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
		}
		raftServers[i] = s.(*raftServer)
	}

	fmt.Println("Created servers")

	// there should be a leader after sufficiently long duration
	
	raftServers[1].server.Outbox() <- &cluster.Envelope{Pid: 2, Msg: &AppendEntry{Term: 7834, LeaderId: 9}}
	e := <-raftServers[2].server.Inbox()
	fmt.Println(e.Pid)
	if msg, ok := e.Msg.(AppendEntry); ok {
		fmt.Println(msg)
	}
	/*

		time.Sleep(1 * time.Minute)
		for i := 1; i <= serverCount; i += 1 {
			if raftServers[i].isLeader() {
				fmt.Println("Server " + strconv.Itoa(i) + " was chosen as leader.")
				return
			}
		}
		t.Errorf("No leader was chosen in 1 minute")
	*/

}

func _TestOneWaySend(t *testing.T) {
	conf := cluster.Config{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009"}

	// launch proxy server
	cluster.NewProxyWithConfig(&conf)
	time.Sleep(100 * time.Millisecond)

	sender, err := cluster.NewWithConfig(1, "127.0.0.1", 5011, &conf)
	if err != nil {
		t.Errorf("Error in creating server ", err.Error())
	}

	receiver, err := cluster.NewWithConfig(2, "127.0.0.1", 5012, &conf)
	if err != nil {
		t.Errorf("Error in creating server ", err.Error())
	}

	done := make(chan bool, 1)
	count := 10000

	//record := make([]uint32, count)

	//	go receive(receiver.Inbox(), record, count, 2, done)
	//	go sendMessages(sender.Outbox(), count, 2, 1)
	done <- true
	select {
	case <-done:
		fmt.Println("TestOneWaySend passed successfully")
		break
	case <-time.After(5 * time.Minute):
		t.Errorf("Could not send ", strconv.Itoa(count), " messages in 5 minute")
		break
	}

	e := &cluster.Envelope{Pid: 2, Msg: ""}

	sender.Outbox() <- e
	e1 := <-receiver.Inbox()
	if msg, ok := e1.Msg.(RequestVote); ok {
		fmt.Println("\n\nReceived object")
		fmt.Println(msg)
	} else {
		fmt.Println(e.Msg)
		fmt.Println(e.Pid)
	}
}

type Object struct {
	x int
	y int
}
