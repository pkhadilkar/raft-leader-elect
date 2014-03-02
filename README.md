Raft Leader Election
=====================

Raft Leader election implements leader election component of [Raft](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf). Raft servers use [cluster](http://github.com/pkhadilkar/cluster) service to send and receive messages. This project currently includes only election component and not log messages. The server term and voted for information is persisted on stable storage.

Installation
-------------
```
$ go get github.com/pkhadilkar/raft-leader-election
$ # cd into directory containing raft-leader-election
$ mkdir logs
$ mkdir stable
$ go test
```
If you want to create logs and stable directories at different path, please change paths RaftConfig object in test files.
Please see installation instructions for [cluster](http://github.com/pkhadilkar/cluster) as cluster uses [ZeroMQ](http://zeromq.org/) libs and [zmq4](https://github.com/pebbe/zmq4) by pebbe.

Test
----

+ **Simple Server Election (leader_test.go)**:
Launches 5 Raft servers and checks that a leader has been elected after sufficiently long time.

+ **Leader Partition and Rejoining (leaderSeparation_test.go)**:
This test case first waits for Raft to elect a leader. It then simulates leader crash by ignoring all messages to/from leader and checks that the remaining servers elect a new leader. It then re-introduces old leader into the cluster and checks that the old leader reverts back to the follower.

+ **Majority Minority Partition (part_test.go)** :
This test case first waits for Raft to elect a leader. It then creates a minority network partition that contains leader and a follower and checks that the servers in majority partition elect a new leader.

+ ** Recovery and Reading state from disk (pstate_test.go)** :
Creates a persistent state file for one of the servers on disk before creating servers. The current term in this state is much higher than default start term. Thus, the expected behavior is that the server with orchestrated state wins the election.

Configuration
----------------
Configuration file contains configuration for both cluster and Raft leader election. Please see [cluster](http://github.com/pkhadilkar/cluster) for cluster specific fields. Raft configuration include *TimeoutInMillis* which is base timeout for election and *HbTimeoutInMillis* - timeout to send periodic heart beats. [Raft](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf) recommends that general relation between timeouts should be "*msg_send_time <<  election_timeout*". Also heartbeat time should be much less than election timeout to avoid frequent elections. *LogDirectoryPath* contains the path to directory used by servers to create log files. Please ensure that this directory is created before using raft-leader-elect. Each server creates a file with name <server's pid>.log in this directory.

Example
--------------
Please see *leader_test.go* test case which also serves as a simple example of use of raft-leader-elect.