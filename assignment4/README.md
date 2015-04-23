Assignment 3. Raft State Machine
===============================

This assignment focusses on the Raft side, implementing the state machine in isolation. Communication between the servers is implemented using channels, will be later replaced by sockets. The server configurations are read from servers.json. Goroutines are spawned acting as servers, Leader Election and Log Replication take place amongst the servers. 

### Testing Instructions

* Run the following on your terminal :

  `go get github.com/abhishekvp/cs733/assignment3/`

  `go test github.com/abhishekvp/cs733/assignment3/spawnServers`


#### TO DO
* ~~Implement Safe Leader Election~~
* ~~Implement Log Replication~~
* ~~Add Test Case for Leader Election~~
* Add More test cases - Leader Change, Message Drop, Log Replication



  

