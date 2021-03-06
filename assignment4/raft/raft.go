package raft

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

/*
* Referred raft Package at https://github.com/pankajrandhe/assignment2/raft
* <Pankaj Randhe and I were group-mates for Assignment 2>
*
 */

//Log sequence number, unique for all time.
type Lsn uint64

// See Log.Append. Implements Error interface.
type ErrRedirect int

type LogEntry interface {
	Lsn() Lsn
	Data() string
	Committed() bool
}

/*
* Map containing all raft instances.
* Used to communicate between instances(servers) through the event channel
* Made Map Concurrency Safe by using sync.RWMutex
 */

var ServersMap = make(map[int]Raft)

// Raft setup
type ServerConfig struct {
	Id         int    // Id of server. Must be unique
	Hostname   string // name or ip of host
	ClientPort int    // port at which server listens to client messages.
	LogPort    int    // tcp port for inter-replica protocol messages.
}

type ClusterConfig struct {
	Path    string         // Directory for persistent log
	Servers []ServerConfig // All servers in this cluster
}

// Raft implements the SharedLog interface.
// Raft Struct
type Raft struct {
	Cluster      *ClusterConfig
	ThisServerId int
	LeaderId     int
	ServersCount int
	currentTerm  int
	votedFor     int
	log          map[Lsn]LogStruct
	EncoderMap   map[int]*gob.Encoder
	commitIndex  int
	lastApplied  int
	nextIndex    []int
	matchIndex   []int
	responses    map[Lsn]int
	EventCh      chan Event
	CommitCh     chan LogStruct
}

type LogStruct struct {
	Log_lsn    Lsn
	Log_data   string
	Log_commit bool
	term       int
}

type VoteReq struct {
	currentTerm  int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}

type VoteRes struct {
	term        int
	voteGranted bool
}

/*
* evType = Type of the event
* 		 : ClientAppend
		 : AppendRPC
		 : Timeout
		 : VoteRequest
		 : VoteResponse
* payload = Message containing Vote or AppendEntries
*/
type Event struct {
	evType  string
	payload interface{}
}

// To be used by AppendRPC
type AppendEntriesReq struct {
	term         int
	leaderId     int
	prevLogIndex int
	prevLogTerm  int
	entry        string
	leaderCommit int
}

type AppendEntriesRes struct {
	term     int
	success  bool
	index    Lsn
	serverId int
}

// Return Values to be used in Follower(), Candidate() and Leader()
const (
	follower  int = 1
	candidate int = 2
	leader    int = 3
)

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(clusterConfig *ClusterConfig, thisServerId int, LeaderId int, CommitCh chan LogStruct) (Raft, error) {
	serversCount := 0
	currentTerm := 0
	votedFor := -1
	logMap := make(map[Lsn]LogStruct)
	EncoderMap := make(map[int]*gob.Encoder)
	commitIndex := -1
	lastApplied := -1
	nextIndex := []int{1, 1, 1, 1, 1}
	matchIndex := []int{0, 0, 0, 0, 0}
	responses := make(map[Lsn]int)
	EventCh := make(chan Event)
	//CommitCh := make(chan LogStruct)

	for _, _ = range clusterConfig.Servers {
		serversCount = serversCount + 1
	}

	var raft = Raft{clusterConfig, thisServerId, LeaderId, serversCount, currentTerm, votedFor, logMap, EncoderMap, commitIndex, lastApplied, nextIndex, matchIndex, responses, EventCh, CommitCh}
	log.Println("Server " + strconv.Itoa(thisServerId) + " Booted!")
	var err error = nil
	//Populating the Common Map containing all raft instance with the newly created raft instance
	//log.Println("Server " + strconv.Itoa(thisServerId) + "New Raft Write Lock Obtained")
	ServersMap[thisServerId] = raft
	//log.Println("Server " + strconv.Itoa(thisServerId) + "New Raft Write Lock Released")
	return raft, err
}

func (raft Raft) Loop(wg sync.WaitGroup) {

	//Server-Server Communication Logic - Referred raft Package authored by Pankaj Randhe

	//Listen on own Server(Log) Port for other Servers
	tcpAddr, connError := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].LogPort))
	if connError != nil {
		log.Fatal(connError)
	}
	listener, connError := net.ListenTCP("tcp", tcpAddr)
	if connError != nil {
		log.Fatal(connError)
	}

	//Register the structs to be sent through socket communication(encoded and decoded)
	gob.Register(VoteReq{})
	gob.Register(VoteRes{})
	gob.Register(AppendEntriesReq{})
	gob.Register(AppendEntriesRes{})
	gob.Register(Event{})

	//Keep Listening for incoming Server Connections - Go Routine - Infinite For Loop
	wg.Add(1)
	go func() {
		for {
			serverConn, connError := listener.Accept()
			log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ": Got a incoming Server Connection. Now Decoding")
			clientInp1 := make([]byte, 512)
			size, _ := serverConn.Read(clientInp1[0:])
			clientInp1 = clientInp1[:size]
			log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + "GOT DATA=")
			log.Println(clientInp1)
			if connError != nil {
				log.Fatal(connError)
			}
			//For each connection, established, decode using gob the message sent by that server
			wg.Add(1)
			go func() {
				log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Entered Decoding Goroutine")
				Decoder := gob.NewDecoder(serverConn)
				log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Created Decoder Object")
				for {
					var Rxdevent Event
					decodeErr := Decoder.Decode(&Rxdevent)
					log.Println(decodeErr)
					if decodeErr == nil {
						log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Decoded Message = ")
						log.Println(Rxdevent)
						//Send the received decoded |Event| Struct onto |EventCh| channel
						raft.EventCh <- Rxdevent
						log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Put in Channel")
					} else {
						log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Decoding Error occured")
					}
				}
				defer serverConn.Close()

			}()
		}
		defer wg.Done()
	}()

	//Connect to  all servers, except self.
	log.Println("S" + strconv.Itoa(raft.Cluster.Servers[raft.ThisServerId].Id) + ":Connect to  all servers, except self.")
	//Populate self's EncoderMap with the encoder object corresponding to other servers
	for _, server := range raft.Cluster.Servers {
		var connRaft net.Conn
		if server.Id != raft.ThisServerId {
			for {
				conn, err := net.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
				if err != nil {
					continue
				} else {
					connRaft = conn
					log.Println("S" + strconv.Itoa(raft.ThisServerId) + ":Dialed connection with : S" + strconv.Itoa(server.Id) + " at Port:" + strconv.Itoa(server.LogPort))
					break
					//Break the infinite loop, with the conn object
				}
			}
			encoder := gob.NewEncoder(connRaft)
			raft.EncoderMap[server.Id] = encoder
		} else {
			raft.EncoderMap[server.Id] = nil
		}
	}

	wg.Add(1)
	go func() {

		for {
			//Keep Checking for Log Entry on the Commit Channel
			data := <-raft.CommitCh
			log.Println("Got DATA", data)
		}
	}()
	defer wg.Done()
	state := follower // begin life as a follower
	for {
		switch state {
		case follower:
			state = raft.Follower()
		case candidate:
			state = raft.Candidate()
		case leader:
			state = raft.Leader()
		default:
			return
		}
	}

}

//Function to ensure that common Map contains all raft instances
//For Debugging Only
func (raft Raft) PrintAllRafts() {
	totalServers := raft.ServersCount
	for i := 0; i < totalServers; i++ {
		fmt.Println(ServersMap[i])
	}

}

func (raft Raft) Follower() int {
	log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": In Follower")
	raft.votedFor = -1

	timer := time.AfterFunc(time.Duration(random(0, 1000))*time.Millisecond, func() {
		log.Print("F S" + strconv.Itoa(raft.ThisServerId) + " Timed Out")
		raft.Send(raft.ThisServerId, Event{"Timeout", nil})
		//raft.EventCh <- Event{"Timeout", nil}
	})

	for {
		event := <-raft.EventCh
		switch event.evType {
		case "ClientAppend":
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Command, I am a Follower.")
			// Do not handle clients in follower mode. Send it back up the
			// pipe with committed = false
			//ev.logEntry.commited = false
			//commitCh <- ev.logentry
		case "VoteRequest":
			msg := event.payload
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Request from S" + strconv.Itoa(msg.(VoteReq).candidateId) + " , I am a Follower.")
			log.Println("F VoteReq Term:" + strconv.Itoa(msg.(VoteReq).currentTerm) + " CurrentTerm:" + strconv.Itoa(raft.currentTerm))
			if msg.(VoteReq).currentTerm < raft.currentTerm {
				//Candidate Term is lesser than Follower Term, hence negative vote
				VoteResInst := VoteRes{raft.currentTerm, false}
				//MapStruct.RLock()
				//log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Obtained")
				//MapStruct.ServersMap[msg.(Vote).OrgServerId].EventCh <- Event{"VoteResponse", Vote{raft.ThisServerId, msg.(Vote).DestServerId, false, raft.currentTerm}}
				raft.Send(msg.(VoteReq).candidateId, Event{"VoteResponse", VoteResInst})
				//MapStruct.RUnlock()
				//log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Released")

				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Sent Negative Vote Response to S" + strconv.Itoa(msg.(VoteReq).candidateId))

			} else {
				//Candidate Term is greater or equal to the Follower Term, requires more scrutiny for voting
				//MapStruct.RLock()
				//Decide whether to vote or not
				voteBool := raft.decideVote(msg.(VoteReq))
				//log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Obtained")
				//MapStruct.ServersMap[msg.(Vote).OrgServerId].EventCh <- Event{"VoteResponse", Vote{raft.ThisServerId, msg.(Vote).DestServerId, true, raft.currentTerm}}
				raft.Send(msg.(VoteReq).candidateId, Event{"VoteResponse", VoteRes{raft.currentTerm, voteBool}})
				//MapStruct.RUnlock()
				//log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Released")
				if voteBool == true {
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Sent Positive Vote Response to S" + strconv.Itoa(msg.(VoteReq).candidateId))
					raft.votedFor = msg.(VoteReq).candidateId
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Voted For = " + strconv.Itoa(raft.votedFor))
					//Reset Election Timeout
					_ = timer.Reset(time.Duration(random(0, 1000)) * time.Millisecond)
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Reset Follower Timer")
				} else {
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Sent Negative Vote Response to S" + strconv.Itoa(msg.(VoteReq).candidateId))
				}
			}

			//if not already voted in my term
			//    reset timer
			//   reply ok to event.msg.serverid
			//    remember term, leader id (either in log or in separate file)
		case "AppendRPC":
			msg := event.payload
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Rxd AppendRPC")
			log.Println(msg.(AppendEntriesReq).entry)

			//Reset Election Timeout
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Reset Election Timeout")
			_ = timer.Reset(time.Duration(random(0, 1000)) * time.Millisecond)

			var RaftLastLogTerm int

			if msg.(AppendEntriesReq).entry != "nil" {
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Message not nil at Follower")
				RaftLastLogIndex := Lsn(len(raft.log))

				//Case when raft.log is empty
				_, exist := raft.log[Lsn(msg.(AppendEntriesReq).prevLogIndex)]
				if exist == true {
					RaftLastLogTerm = raft.log[Lsn(msg.(AppendEntriesReq).prevLogIndex)].term
				} else {
					RaftLastLogTerm = 0
				}
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Current Term = " + strconv.Itoa(raft.currentTerm))
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Sender Term = " + strconv.Itoa(msg.(AppendEntriesReq).term))

				if msg.(AppendEntriesReq).term < raft.currentTerm {

					//Current Term is greater than Leader Term, send false
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Current Term is greater than Leader Term, send false")
					AppendEntriesResInst := AppendEntriesRes{raft.currentTerm, false, RaftLastLogIndex, raft.ThisServerId}
					//MapStruct.RLock()
					raft.Send(msg.(AppendEntriesReq).leaderId, Event{"AppendRPCRes", AppendEntriesResInst})
					//MapStruct.RUnlock()
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "sent false to AppendRPC from leader")

				} else {
					raft.currentTerm = msg.(AppendEntriesReq).term
				}
				if msg.(AppendEntriesReq).prevLogTerm != RaftLastLogTerm {

					// Follower's Log Term mismatch with that of AppendEntries Prev Log Term
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower's Log Term mismatch with that of AppendEntries Prev Log Term")
					AppendEntriesResInst := AppendEntriesRes{raft.currentTerm, false, RaftLastLogIndex, raft.ThisServerId}
					//MapStruct.RLock()
					raft.Send(msg.(AppendEntriesReq).leaderId, Event{"AppendRPCRes", AppendEntriesResInst})
					//	MapStruct.RUnlock()
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "sent false to AppendRPC from leader")

				} else {

					// Follower's log is identical to that of the Leader. Safe to push new entries into the Follower's Log
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower's log is identical to that of the Leader. Safe to push new entries into the Follower's Log")
					//Insert the log entry in to the map |log|
					logStructInst := LogStruct{RaftLastLogIndex + 1, msg.(AppendEntriesReq).entry, false, msg.(AppendEntriesReq).term}
					raft.log[RaftLastLogIndex+1] = logStructInst
					AppendEntriesResInst := AppendEntriesRes{raft.currentTerm, true, RaftLastLogIndex, raft.ThisServerId}
					//MapStruct.RLock()
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "RaftLastLogIndex =" + strconv.Itoa(int(RaftLastLogIndex)))
					raft.Send(msg.(AppendEntriesReq).leaderId, Event{"AppendRPCRes", AppendEntriesResInst})
					//MapStruct.RUnlock()
					//Code to write to log FILE
					raft.LogToDisk(msg.(AppendEntriesReq).entry)
					raft.LogToDisk("\n")
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "sent true to AppendRPC from leader")

				}

			} else {

				//HeartBeat
				log.Println("HEARTBEAT! HEARTBEAT HEARTBEAT HEARTBEAT!")
				if raft.currentTerm < msg.(AppendEntriesReq).term {
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower updated TERM to leader TERM")
					raft.currentTerm = msg.(AppendEntriesReq).term
				}
				raft.LeaderId = msg.(AppendEntriesReq).leaderId
				if msg.(AppendEntriesReq).leaderCommit > raft.commitIndex {
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Self Commit Index = " + strconv.Itoa(raft.commitIndex))
					if msg.(AppendEntriesReq).leaderCommit < len(raft.log) {
						raft.commitIndex = msg.(AppendEntriesReq).leaderCommit
					} else {
						raft.commitIndex = len(raft.log)
					}
					log.Println("Leader Commit Detected. Updated self commitIndex to " + strconv.Itoa(raft.commitIndex))
					raft.CommitCh <- raft.log[Lsn(raft.commitIndex)]
				}
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Rxd HeartBeat from Leader S" + strconv.Itoa(raft.LeaderId))

			}

		case "Timeout":
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Timed Out. Now Candidate")
			timer.Stop()
			return candidate // new state back to loop()
		}

	}

	return 0
}

func (raft Raft) Leader() int {
	log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " Appointed LEADER!")

	// HeatBeat/AppendRPCEntry Sending Logic
	go func() {
		for {
			var data string
			for _, server := range raft.Cluster.Servers {
				//log.Println("Inside servers For loop")
				var RaftLastLogTerm int
				//Workaround Go Lang Issue
				tempRaftInst := ServersMap[server.Id]
				tempRaftInst.LeaderId = raft.ThisServerId
				ServersMap[server.Id] = tempRaftInst
				//log.Println("In inside for loop")
				if server.Id != raft.ThisServerId {
					log.Println("L S" + strconv.Itoa(raft.ThisServerId) + ": Len of my Log = " + strconv.Itoa(len(raft.log)))
					log.Println("L S" + strconv.Itoa(raft.ThisServerId) + ": NextIndex[" + strconv.Itoa(server.Id) + "] = " + strconv.Itoa(raft.nextIndex[server.Id]))
					if len(raft.log) >= raft.nextIndex[server.Id] {
						//Log Entry to send
						log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "Log Entry to replicate")
						data = raft.log[Lsn(raft.nextIndex[server.Id])].Log_data
						log.Println(data)
						raft.nextIndex[server.Id] = raft.nextIndex[server.Id] + 1

					} else {
						//HeartBeat to send
						log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "HeartBeat to send")
						data = "nil"
					}
					//Case when raft.log is empty
					_, exist := raft.log[Lsn(len(raft.log))]
					if exist == true {
						RaftLastLogTerm = raft.log[Lsn(raft.nextIndex[server.Id]-1)].term
					} else {
						RaftLastLogTerm = 0
					}
					//log.Print("L S" + strconv.Itoa(raft.ThisServerId) + " Leader Read Lock Obtained")
					//MapStruct.ServersMap[server.Id].EventCh <- heartBeat
					appendEntry := Event{"AppendRPC", AppendEntriesReq{raft.currentTerm, raft.ThisServerId, raft.nextIndex[server.Id] - 1, RaftLastLogTerm, data, raft.commitIndex}}
					raft.Send(server.Id, appendEntry)
					//log.Print("L S" + strconv.Itoa(raft.ThisServerId) + " Leader Read Lock Released")
					log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " sent AppendRPC to " + strconv.Itoa(server.Id))
				}
			}
			//time.Sleep(time.Duration(5) * time.Millisecond)
		}
	}()

	for {
		//time.Sleep(time.Duration(5) * time.Millisecond)
		event := <-raft.EventCh
		switch event.evType {
		case "ClientAppend":
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Command, But I am a Leader.")
			// Do not handle clients in follower mode. Send it back up the
			// pipe with committed = false
			//ev.logEntry.commited = false
			//commitCh <- ev.logentry

		case "VoteResponse":
			//msg := event.payload
			log.Println("L S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Response") // from S" + strconv.Itoa(msg.(VoteRes).candidateId) + ", I am a Candidate")

		case "VoteRequest":
			//msg := event.payload
			log.Println("L Rxd VoteRequest") // + strconv.Itoa(msg.(VoteReq).candidateId))
			/*
				if msg.(Vote).OrgServerTerm <= raft.currentTerm {
					heartBeat := Event{"AppendRPC", nil}
					MapStruct.Lock()
					fmt.Print("L S" + strconv.Itoa(raft.ThisServerId) + "Leader Write Lock Obtained")
					MapStruct.ServersMap[msg.(Vote).OrgServerId].EventCh <- heartBeat
					MapStruct.Unlock()
					fmt.Print("L S" + strconv.Itoa(raft.ThisServerId) + "Leader Write Lock Released")
					log.Print("L Sent HBT to S"+strconv.Itoa(msg.(Vote).OrgServerId))

				}
			*/
		case "AppendRPCRes":
			msg := event.payload
			if msg.(AppendEntriesRes).success {
				log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "Received Successfull AppendRPC Response from S" + strconv.Itoa(msg.(AppendEntriesRes).serverId))

				raft.matchIndex[msg.(AppendEntriesRes).serverId] = int(msg.(AppendEntriesRes).index)
				log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "Original Next Index[" + strconv.Itoa(raft.ThisServerId) + "] =" + strconv.Itoa(raft.nextIndex[msg.(AppendEntriesRes).serverId]))
				//raft.nextIndex[msg.(AppendEntriesRes).serverId] = raft.nextIndex[msg.(AppendEntriesRes).serverId] + 1
				log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "Updated Next Index[" + strconv.Itoa(raft.ThisServerId) + "] =" + strconv.Itoa(raft.nextIndex[msg.(AppendEntriesRes).serverId]))

				//To count the number of responses - number of servers that have replication the log with index = msg.(AppendEntriesRes).index
				raft.responses[Lsn(msg.(AppendEntriesRes).index)] = int(msg.(AppendEntriesRes).index) + 1
				log.Println("L S" + strconv.Itoa(raft.ThisServerId) + "Number of Responses = " + strconv.Itoa(raft.responses[Lsn(msg.(AppendEntriesRes).index)]))
				if raft.responses[Lsn(msg.(AppendEntriesRes).index)] == 2 {
					//Execute on the State Machine
					raft.CommitCh <- raft.log[Lsn(msg.(AppendEntriesRes).index)]
					log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " Log Entry =")
					log.Println(raft.log[Lsn(msg.(AppendEntriesRes).index)])
					raft.commitIndex = int(msg.(AppendEntriesRes).index)
					log.Println("Updated Leader Commit Index")
				}
			}

		}

	}
	return 0
}

func (raft Raft) Candidate() int {
	log.Println("S" + strconv.Itoa(raft.ThisServerId) + ": In Candidate")

	raft.currentTerm = raft.currentTerm + 1

	//Self Vote
	votesRxd := 1

	timer := time.AfterFunc(time.Duration(random(0, 1000))*time.Millisecond, func() {
		log.Print("C S" + strconv.Itoa(raft.ThisServerId) + " Timed Out")
		//raft.EventCh <- Event{"Timeout", nil}
		raft.Send(raft.ThisServerId, Event{"Timeout", nil})
	})

	//Send VoteRequest to all servers
	go func() {
		for _, server := range raft.Cluster.Servers {
			if server.Id != raft.ThisServerId {
				//	MapStruct.RLock()
				//log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Candidate Write Lock Obtained")
				//MapStruct.ServersMap[server.Id].EventCh <- Event{"VoteRequest", Vote{raft.ThisServerId, server.Id, false, raft.currentTerm}}

				//Compute LastLogIndex and LastLogTerm
				lastLogIndex := len(raft.log)
				log.Println("lastLogIndex = " + strconv.Itoa(lastLogIndex))

				var lastLogTerm int

				if lastLogIndex != 0 {
					lastLogTerm = raft.log[Lsn(lastLogIndex)].term
				} else { //Case when the map |log| is empty
					lastLogTerm = 0
				}

				log.Println("lastLogTerm = " + strconv.Itoa(lastLogTerm))

				VoteReqInst := VoteReq{raft.currentTerm, raft.ThisServerId, lastLogIndex, lastLogTerm}

				log.Print("VoteReqInst = ")
				log.Println(VoteReqInst)

				raft.Send(server.Id, Event{"VoteRequest", VoteReqInst})
				//MapStruct.RUnlock()
				//log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Candidate Write Lock Released")
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + " Sending vote request to " + strconv.Itoa(server.Id))

			}
		}
	}()

	for {

		event := <-raft.EventCh
		log.Println("C Listening Loop : S" + strconv.Itoa(raft.ThisServerId))
		log.Println(event)
		switch event.evType {
		case "ClientAppend":
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Command, But I am a Candidate.")
			// Do not handle clients in follower mode. Send it back up the
			// pipe with committed = false
			//ev.logEntry.commited = false
			//commitCh <- ev.logentry
		case "VoteResponse":
			msg := event.payload
			//log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Response from S" + strconv.Itoa(msg.(VoteReq).candidateId) + ", I am a Candidate")
			if msg.(VoteRes).voteGranted == true {
				votesRxd = votesRxd + 1
				log.Println("C Votes = " + strconv.Itoa(votesRxd))
				if votesRxd > raft.ServersCount/2 {
					return leader
				}
			}
		case "VoteRequest":
			msg := event.payload
			if msg.(VoteReq).currentTerm >= raft.currentTerm {
				raft.currentTerm = msg.(VoteReq).currentTerm
				raft.votedFor = -1
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Request from S" + strconv.Itoa(msg.(VoteReq).candidateId) + " , Becoming Follower")
				return follower
			}

		case "AppendRPC":
			msg := event.payload
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Rxd AppendRPC")

			//Reset Election Timeout
			//log.Println("Reset Election Timeout")

			raft.currentTerm = msg.(AppendEntriesReq).term

			log.Println("C HB at S" + strconv.Itoa(raft.ThisServerId) + " Becoming Follower")
			timer.Stop()
			return follower

			// reset timer
			// if msg.term < currentterm, ignore
			// reset heartbeat timer
			// upgrade to event.msg.term if necessary
			// if prev entries of my log and event.msg match
			//    add to disk log
			//   flush disk log
			//   respond ok to event.msg.serverid
			//else
			//  respond err.
		case "Timeout":
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Candidate Timed Out. Resetting")
			timer.Stop()
			return candidate
		}
	}

	return 0
}

//Reference : Prof. Sriram's comment on Piazza
func (raft Raft) Send(toServerId int, msg Event) {

	/*
		r := random(0, 100)
		delay := 10
		if r <= 50 {
			toServer.Receive(msg)
		} else if r > 50 {
			// delayed send
			time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
				toServer.Receive(msg)
			})
		}//else do nothing. Msg dropped
	*/
	//Send using sockets
	log.Println("Send: Send to " + strconv.Itoa(toServerId))
	encoder := raft.EncoderMap[raft.Cluster.Servers[toServerId].Id]
	if encoder != nil {
		log.Println("Send: Got encoder object")
		_ = encoder.Encode(msg)
		log.Println("Send: Encoded Message =")
		log.Println(msg)
	} else {
		log.Println("Send: Did not get encoder obj, Now filling in EncoderMap")
		var connRaft net.Conn
		for {
			conn, err := net.Dial("tcp", ":"+strconv.Itoa(raft.Cluster.Servers[toServerId].LogPort))
			conn.Write([]byte("Hello"))
			log.Println("Send: Sent Hello to " + strconv.Itoa(raft.Cluster.Servers[toServerId].Id))
			if err != nil {
				continue
			} else {
				connRaft = conn
				break
				//Break the infinite loop, with the conn object
			}
		}
		encoder := gob.NewEncoder(connRaft)
		raft.EncoderMap[raft.Cluster.Servers[toServerId].Id] = encoder
		log.Println("Send: Encoder Obj Reattempted to put in EncMap")
		//Now Send the data
		encoder.Encode(msg)
		log.Println("Send: Encoded Message =")
		log.Println(msg)
		Decoder := gob.NewDecoder(connRaft)
		log.Println("Send: Decoder Created in Send")
		var Rxdevent Event
		decodeErr := Decoder.Decode(&Rxdevent)
		log.Println("Send: Error=")
		log.Println(decodeErr)
		if decodeErr == nil {
			log.Println(" Send:Decoded Message = ")
			log.Println(Rxdevent)
		}

	}

}

//Not Used for Assignment 4
func (raft Raft) Receive(msg Event) {
	raft.EventCh <- msg
}

//Decides whether to grant vote or not.
func (raft Raft) decideVote(vote VoteReq) bool {

	var rxLastLogTerm int

	if raft.votedFor == -1 || raft.votedFor == vote.candidateId {

		//MapStruct.RLock()
		rxRaft := ServersMap[raft.ThisServerId]
		//MapStruct.RUnlock()
		//Decide whose log is more up-to-date

		_, exist := rxRaft.log[Lsn(len(rxRaft.log))]
		if exist == true {
			rxLastLogTerm = rxRaft.log[Lsn(len(rxRaft.log))].term
		} else {
			//Case when log map is empty
			rxLastLogTerm = 0
		}
		rxLastLogIndex := len(rxRaft.log)

		if rxLastLogTerm > vote.lastLogTerm {
			return false
		} else if rxLastLogTerm < vote.lastLogTerm {
			return true
		} else {
			//rxLastLogTerm is equal to vote.lastLogTerm
			//Check for latest log index
			if rxLastLogIndex <= vote.lastLogIndex {
				return true
			} else {
				return false
			}
		}

	}

	return false

}

func (x LogStruct) Lsn() Lsn {

	return x.Log_lsn
}

func (x LogStruct) Data() string {

	return x.Log_data
}

func (x LogStruct) Committed() bool {

	return x.Log_commit
}

type SharedLog interface {
	// Each data item is wrapped in a LogEntry with a unique
	// lsn. The only error that will be returned is ErrRedirect,
	// to indicate the server id of the leader. Append initiates
	// a local disk write and a broadcast to the other replicas,
	// and returns without waiting for the result.
	Append(data []byte) (LogEntry, error)
}

//Src : http://golangcookbook.blogspot.in/2012/11/generate-random-number-in-given-range.html
func random(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "ERR_REDIRECT " //+ raft.Cluster.Servers[int(e)].Hostname + " " + strconv.Itoa(raft.Cluster.Servers[int(e)].ClientPort) + "\r\n"
}

func (raft Raft) Append(data string) (LogEntry, error) {

	if raft.ThisServerId != raft.LeaderId {
		return nil, ErrRedirect(raft.LeaderId)

	} else {

		// Prepare the LogEntry
		var log_instance LogStruct
		lsn := Lsn(len(raft.log) + 1)
		log_instance = LogStruct{lsn, data, false, raft.currentTerm}

		//Initialize number of ack for the lsn to 0
		//string_lsn := strconv.Itoa(int(log_instance.Lsn()))

		//Push the log entry to the |log| map
		raft.log[lsn] = log_instance
		log.Println("Append : Log Entry pushed to Log Map of Leader")
		log.Println("Data at Leader = " + raft.log[lsn].Log_data)

		/*

			// Take the raft object and broadcast the log-entry
			// Lets send the logentry to each of the servers in the cluster

			//Read back the servers as JSON objects
			servers := raft.Cluster.Servers

			for _, server := range servers {

				//host := server.Hostname
				port := server.LogPort

				//now establish the TCP connection and send the data to the follower servers
				connection, err := net.Dial("tcp", ":"+strconv.Itoa(port))
				if err != nil {
					continue
				} else {
					_, _ = connection.Write([]byte(strconv.Itoa(int(log_instance.Lsn())) + " " + string(log_instance.Data()) + " false"))
				}
			}*/
		// Prepare the log entry and return it
		return log_instance, nil
	}
}

func (raft Raft) LogToDisk(data string) {
	/*
	* Write the received log entry - data byte to a file in the local disk
	*
	* References:
	* Writing to Files - https://gobyexample.com/writing-files
	* Appending to Files - http://stackoverflow.com/questions/7151261/append-to-a-file-in-go?lq=1
	* Check whether file already exists - http://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
	 */

	filename := (raft.Cluster).Path + strconv.Itoa(raft.ThisServerId)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
	}

	logFile, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	defer logFile.Close()

	if _, err = logFile.WriteString(data); err != nil {
		panic(err)
	}
}
