Raft Leader Election
=====================

Raft Leader election implements leader election component of [Raft](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf). Raft servers use [cluster](http://github.com/pkhadilkar/cluster) service to send and receive messages. This project currently includes only election component and not log messages.

Installation
-------------
```
$ go get github.com/pkhadilkar/raft-leader-election
$ # cd into directory containing raft-leader-election
$ go test
```

Please see installation instructions for [cluster](http://github.com/pkhadilkar/cluster) as cluster uses [ZeroMQ](http://zeromq.org/) libs and [zmq4](https://github.com/pebbe/zmq4) by pebbe.

Test
----

+ **Simple Server Election (leader_test.go)**:
This test case launches 5 Raft servers and checks that a leader has been elected after sufficiently long time.

+ **Leader Partition and Rejoining (leaderSeparation_test.go)**:
This test case first waits for Raft to elect a leader. It then simulates leader crash by ignoring all messages to/from leader and checks that the remaining servers elect a new leader. It then re-introduces old leader into the cluster and checks that the old leader reverts back to the follower.

+ **Majority Minority Partition (part_test.go)** :
This test case first waits for Raft to elect a leader. It then creates a minority network partition that contains leader and a follower and checks that the servers in majority partition elect a new leader.

Configuration
----------------
Configuration file contains configuration for both cluster and Raft leader election. Please see [cluster](http://github.com/pkhadilkar/cluster) for cluster specific fields. Raft configuration include *TimeoutInMillis* which is base timeout for election and *HbTimeoutInMillis* - timeout to send periodic heart beats. [Raft](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf) recommends that general relation between timeouts should be "*msg_send_time <<  election_timeout*". Also heartbeat time should be much less than election timeout to avoid frequent elections.

Example
--------------
Please see *leader_test.go* test case which also serves as a simple example of use of raft-leader-elect.