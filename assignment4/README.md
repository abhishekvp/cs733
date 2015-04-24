Assignment 4. Raft Consensus Algorithm
======================================

In a distributed system of multiple servers, each server needs to agree to the same data. Raft Consensus Algorithm proposes a log based approach to have consensus amongst the servers.

###File Structure
* raft/raft.go - Raft Package
* spawnServer/spawnServer.go
	- initialises a Raft Instance, the KV Store and the Client Handler
* spawnServer/kv.go
	- KV Store
* runner/runner.go
	- Reads the server cluster config from servers.json file and calls spawnServer binary, thus spawning 5 server processes

### Usage Instructions

Run the following commands:

`go get github.com/abhishekvp/cs733/assignment4`

`go build github.com/abhishekvp/cs733/assignment4/raft`

`go install github.com/abhishekvp/cs733/assignment4/spawnServer`

`go install github.com/abhishekvp/cs733/assignment4/runner`

`runner`

### Bugs
Currently server-server communication fails due to not being able to decode the received messages.


###Program Flow

* Once the servers are up, they start in the Follower State.
* After a random timeout, A Follower becomes a Candidate.
* A Candidate conducts an election, requesting votes from other Followers. On getting a majority of Votes, it becomes the Leader.
* The leader periodically sends Heartbeats to all servers, stating that its the leader in that term.
* A client connects to a server, not knowing which is the Leader. If not the Leader, the server will send a Redirect Error to the Client, giving the port number of the Leader.
* Once the Client is connected to the Leader. It can issue commands to the Leader, in this case KV Commands.
* On receiving a command from a client, the leader does the following:
	- Leader first Appends the log entry in its own log
	- Leader then sends the log entry to all the other servers
	- The servers, check whether their log is consistent with the leader and append the received Log Entry.
	- Once appended, the servers then send a AppendRPC Response to the Leader.
	- If the Leader, receives a majority responses for that entry, it marks the entry as committed.
	- It then sends the entry to its KV Store (State Machine) for execution and replies back to the client.
	- All the other servers detect that the leader committed the entry, and they to send the entry to their respective state machines.




  


