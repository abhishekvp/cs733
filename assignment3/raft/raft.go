package raft

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

/*
* Referred raft Package at https://github.com/pankajrandhe/assignment2/raft
* <Pankaj Randhe and I were group-mates for Assignment 2>
*
* Referred raft Package authored by Pankaj Randhe at https://github.com/pankajrandhe/assignment3/raft
 */

//Log sequence number, unique for all time.
type Lsn uint64

// See Log.Append. Implements Error interface.
type ErrRedirect int

type LogEntry interface {
	Lsn() Lsn
	Data() []byte
	Committed() bool
}

/*
* Map containing all raft instances.
* Used to communicate between instances(servers) through the event channel
 */

var MapStruct = struct {
	sync.RWMutex
	ServersMap map[int]Raft
}{ServersMap: make(map[int]Raft)}

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
	commitIndex  int
	lastApplied  int
	EventCh      chan Event
}

type LogStruct struct {
	Log_lsn    Lsn
	Log_data   []byte
	Log_commit bool
}

type Vote struct {
	OrgServerId   int
	DestServerId  int
	VoteValue     bool
	OrgServerTerm int
}

type Event struct {
	evType  string
	payload interface{}
}

const (
	follower  int = 1
	candidate int = 2
	leader    int = 3
)

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(clusterConfig *ClusterConfig, thisServerId int, LeaderId int) (Raft, error) {
	serversCount := 0
	currentTerm := 0
	votedFor := -1
	commitIndex := -1
	lastApplied := -1
	EventCh := make(chan Event)

	for _, _ = range clusterConfig.Servers {
		serversCount = serversCount + 1
	}

	var raft = Raft{clusterConfig, thisServerId, LeaderId, serversCount, currentTerm, votedFor, commitIndex, lastApplied, EventCh}
	log.Println("Server " + strconv.Itoa(thisServerId) + " Booted!")
	var err error = nil
	//Populating the Common Map containing all raft instance with the newly created raft instance
	MapStruct.Lock()
	log.Println("Server " + strconv.Itoa(thisServerId) + "New Raft Write Lock Obtained")
	MapStruct.ServersMap[thisServerId] = raft
	log.Println("Server " + strconv.Itoa(thisServerId) + "New Raft Write Lock Released")
	MapStruct.Unlock()
	return raft, err
}

func (raft Raft) Loop(wg sync.WaitGroup) {
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
		MapStruct.RLock()
		fmt.Println(MapStruct.ServersMap[i])
		MapStruct.RUnlock()
	}

}

func (raft Raft) Follower() int {
	log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": In Follower")
	timer := time.AfterFunc(time.Duration(random(0, 1000))*time.Millisecond, func() {
		log.Print("F S" + strconv.Itoa(raft.ThisServerId) + " Timed Out")
		raft.EventCh <- Event{"FTimeout", nil}
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
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Request from S" + strconv.Itoa(msg.(Vote).OrgServerId) + " , I am a Follower.")
			log.Println("F VoteReq Term:" + strconv.Itoa(msg.(Vote).OrgServerTerm) + " CurrentTerm:" + strconv.Itoa(raft.currentTerm))
			if msg.(Vote).OrgServerTerm < raft.currentTerm {
				MapStruct.Lock()
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Obtained")
				MapStruct.ServersMap[msg.(Vote).OrgServerId].EventCh <- Event{"VoteResponse", Vote{raft.ThisServerId, msg.(Vote).DestServerId, false, raft.currentTerm}}
				MapStruct.Unlock()
				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Released")

				log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Sent Negative Vote Response to S" + strconv.Itoa(msg.(Vote).OrgServerId))
				//Reset Election Timeout
				_ = timer.Reset(time.Duration(random(0, 1000)) * time.Millisecond)
			}
			if msg.(Vote).OrgServerTerm > raft.currentTerm {
				raft.currentTerm = msg.(Vote).OrgServerTerm
				if raft.votedFor == -1 {
					MapStruct.Lock()
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Obtained")
					MapStruct.ServersMap[msg.(Vote).OrgServerId].EventCh <- Event{"VoteResponse", Vote{raft.ThisServerId, msg.(Vote).DestServerId, true, raft.currentTerm}}
					MapStruct.Unlock()
					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Write Lock Released")

					log.Println("F S" + strconv.Itoa(raft.ThisServerId) + ": Sent Positive Vote Response to S" + strconv.Itoa(msg.(Vote).OrgServerId))
					raft.votedFor = msg.(Vote).OrgServerId
					//Reset Election Timeout
					_ = timer.Reset(time.Duration(random(0, 1000)) * time.Millisecond)
				}
			}
			//if not already voted in my term
			//    reset timer
			//   reply ok to event.msg.serverid
			//    remember term, leader id (either in log or in separate file)
		case "AppendRPC":
			msg := event.payload
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Rxd AppendRPC")
			//HeartBeat from Server, therefore reset timer

			if msg == nil {
				log.Println("F HB at S" + strconv.Itoa(raft.ThisServerId) + ": Resetting self-timer")
				timer.Reset(time.Duration(random(0, 1000)) * time.Millisecond)
				//return follower
				//_ = timer.Reset(time.Duration(random(150,350)) * time.Millisecond)
			}
			// reset timer
			// if msg.term < currentterm, ignore
			// reset heartbeat timer
			// upgrade to event.msg.term if necessary
			// if prev entries of my log and event.msg match
			//    add to disk log
			//   flush disk log
			//   respond ok to event.msg.serverid
			// else
			//  respond err.
		case "FTimeout":
			log.Println("F S" + strconv.Itoa(raft.ThisServerId) + "Follower Timed Out. Now Candidate")
			timer.Stop()
			return candidate // new state back to loop()
		}

	}

	return 0
}

func (raft Raft) Leader() int {
	log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " Appointed LEADER!")

	heartBeat := Event{"AppendRPC", nil}

	go func() {
		for {
			time.Sleep(time.Duration(10) * time.Millisecond)
			for _, server := range raft.Cluster.Servers {

				//Workaround Go Lang Issue
				MapStruct.RLock()
				tempRaftInst := MapStruct.ServersMap[server.Id]
				MapStruct.RUnlock()
				tempRaftInst.LeaderId = raft.ThisServerId
				MapStruct.Lock()
				MapStruct.ServersMap[server.Id] = tempRaftInst
				MapStruct.Unlock()

				if server.Id != raft.ThisServerId {

					log.Println("In inside for loop")
					MapStruct.Lock()
					log.Print("L S" + strconv.Itoa(raft.ThisServerId) + " Leader Read Lock Obtained")
					//log.Println("L S"+strconv.Itoa(raft.ThisServerId)+" sent HeartBeat to "+strconv.Itoa(server.Id))
					MapStruct.ServersMap[server.Id].EventCh <- heartBeat
					MapStruct.Unlock()
					log.Print("L S" + strconv.Itoa(raft.ThisServerId) + " Leader Read Lock Released")

					log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " sent HeartBeat to " + strconv.Itoa(server.Id))

				}
			}
		}

	}()

	for {

		event := <-raft.EventCh
		switch event.evType {
		case "ClientAppend":
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Command, But I am a Leader.")
			// Do not handle clients in follower mode. Send it back up the
			// pipe with committed = false
			//ev.logEntry.commited = false
			//commitCh <- ev.logentry

		case "VoteResponse":
			msg := event.payload
			log.Println("L S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Response from S" + strconv.Itoa(msg.(Vote).OrgServerId) + ", I am a Candidate")

		case "VoteRequest":
			msg := event.payload
			log.Println("L Rxd VoteRequest from S" + strconv.Itoa(msg.(Vote).OrgServerId))
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
		}

		log.Println("L S" + strconv.Itoa(raft.ThisServerId) + " In Leader : infi loop")
		//Send HeartBeat to all servers

	}
	return 0
}

func (raft Raft) Candidate() int {
	log.Println("S" + strconv.Itoa(raft.ThisServerId) + ": In Candidate")

	timer := time.AfterFunc(time.Duration(random(0, 1000))*time.Millisecond, func() {
		log.Print("C S" + strconv.Itoa(raft.ThisServerId) + " Timed Out")
		raft.EventCh <- Event{"CTimeout", nil}
	})

	go func() {
		for _, server := range raft.Cluster.Servers {
			if server.Id != raft.ThisServerId {
				MapStruct.Lock()
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Candidate Write Lock Obtained")
				MapStruct.ServersMap[server.Id].EventCh <- Event{"VoteRequest", Vote{raft.ThisServerId, server.Id, false, raft.currentTerm}}
				MapStruct.Unlock()
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Candidate Write Lock Released")
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + " Sending vote request to " + strconv.Itoa(server.Id))

			}
		}
	}()

	//Self Vote
	votesRxd := 1
	raft.currentTerm = raft.currentTerm + 1

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
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Response from S" + strconv.Itoa(msg.(Vote).OrgServerId) + ", I am a Candidate")
			if msg.(Vote).VoteValue == true {
				votesRxd = votesRxd + 1
				log.Println("C Votes = " + strconv.Itoa(votesRxd))
				if votesRxd > raft.ServersCount/2 {
					return leader
				}
			}
		case "VoteRequest":
			msg := event.payload
			if msg.(Vote).OrgServerTerm >= raft.currentTerm {
				raft.currentTerm = msg.(Vote).OrgServerTerm
				raft.votedFor = -1
				log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Rxd Vote Request from S" + strconv.Itoa(msg.(Vote).OrgServerId) + " , Becoming Follower")
				return follower
			}

		case "AppendRPC":
			msg := event.payload
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + "Rxd AppendRPC")
			//HeartBeat from Server, therefore become follower
			if msg == nil {
				log.Println("C HB at S" + strconv.Itoa(raft.ThisServerId) + " Becoming Follower")
				timer.Stop()
				return follower
			}
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
		case "CTimeout":
			log.Println("C S" + strconv.Itoa(raft.ThisServerId) + ": Candidate Timed Out. Resetting")
			timer.Stop()
			//log.Println(<-raft.EventCh)
			return candidate
		}
	}

	return 0
}

func (x LogStruct) Lsn() Lsn {

	return x.Log_lsn
}

func (x LogStruct) Data() []byte {

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

/*
// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "ERR_REDIRECT " + raft.Cluster.Servers[int(e)].Hostname + " " + strconv.Itoa(raft.Cluster.Servers[int(e)].ClientPort) + "\r\n"
}

func (raft Raft) Append(data []byte) (LogEntry, error) {

	if raft.ThisServerId != raft.LeaderId {

		return nil, ErrRedirect(raft.LeaderId)

	} else {

		// Prepare the LogEntry
		var log_instance LogStruct
		lsn := Lsn(uint64(rand.Int63()))
		log_instance = LogStruct{lsn, data, false}

		//Initialize number of ack for the lsn to 0
		string_lsn := strconv.Itoa(int(log_instance.Lsn()))



		* Write the received log entry - data byte to a file in the local disk
		*
		* References:
		* Writing to Files - https://gobyexample.com/writing-files
		* Appending to Files - http://stackoverflow.com/questions/7151261/append-to-a-file-in-go?lq=1
		* Check whether file already exists - http://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go


		filename := (raft.Cluster).Path

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

		// ########NOTE: DUMMY value for commit, remember to correct it!
		if _, err = logFile.WriteString("\n" + strconv.Itoa(int(log_instance.Log_lsn)) + " " + string(log_instance.Log_data) + "false"); err != nil {
			panic(err)
		}

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
		}
		// Prepare the log entry and return it
		return log_instance, nil
	}
}
*/
