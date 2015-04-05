Assignment 3. Raft State Machine
===============================

This assignment focusses on the Raft side, implementing the state machine in isolation. Communication between the servers is implemented using channels, will be later replaced by sockets. The server configurations are read from servers.json. Goroutines are spawned acting as servers, Leader Election and Log Replication take place amongst the servers. 

### Usage Instructions

* Run the following on your terminal :

  `go get github.com/abhishekvp/cs733/assignment3/`

  `go run github.com/abhishekvp/cs733/assignment3/spawnSMs/spawnSMs.go`



#### TO DO
* ~~Implement Safe Leader Election~~
* Implement Log Replication
* Add Test Cases



  

