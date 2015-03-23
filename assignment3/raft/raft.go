package raft

import (
	"strconv"
	"fmt"
	"math/rand"
)

var raft Raft

type Lsn uint64 //Log sequence number, unique for all time.

type ErrRedirect int // See Log.Append. Implements Error interface.

type LogEntry interface {
	Lsn() Lsn
	Data() []byte
	Committed() bool
}

var ServersMap map[int]Raft

type Vote {
	OrgServerId int
	DestServerId int
	VoteValue bool
	OrgServerTerm int
}

type Event struct {
	evType string
	payload interface{}
}

func (raft Raft) Loop() {
    state := follower; // begin life as a follower

    for {
        switch (state)  {
        case follower: state = raft.follower()
        case candidate: state = raft.candidate()
        case leader: state = raft.leader()
        default: return
        }
    }
}

func (raft Raft) follower() {

    timer := time.NewTimer(time.Second*rand.Intn(10)) // to become candidate if no append reqs

    go func() {
    	for {
    	<- timer.C
    	raft.EventCh = Event{"Timeout",nil}
    	}
    	}()

    for {
        event := <- raft.EventCh
        switch event.(type) {
        case "ClientAppend":
        	fmt.Println("S"+strconv.Itoa(raft.thisServerId) + ": Rxd Command, But I am a Follower. NOACT" )
            // Do not handle clients in follower mode. Send it back up the
            // pipe with committed = false
            //ev.logEntry.commited = false
            //commitCh <- ev.logentry
        case "VoteRequest":
        	msg = event.payload
        	fmt.Println("S"+strconv.Itoa(raft.thisServerId) + ": Rxd Vote Request from , But I am a Follower. NOACT" )
 
            if msg.OrgServerTerm < raft.currentTerm {
            	raft.EventCh = Event{"VoteResponse", Vote{raft.thisServerId, msg.DestServerId, false, raft.currentTerm}}
            }
            else if msg.OrgServerTerm > raft.currentTerm {
            	raft.currentTerm = msg.OrgServerTerm
            }
            //if not already voted in my term
            //    reset timer
             //   reply ok to event.msg.serverid
            //    remember term, leader id (either in log or in separate file)
        case "AppendRPC":
           // reset timer
           // if msg.term < currentterm, ignore
           // reset heartbeat timer
           // upgrade to event.msg.term if necessary
           // if prev entries of my log and event.msg match
           //    add to disk log
            //   flush disk log
            //   respond ok to event.msg.serverid
            else
               respond err.
        case "Timeout": 
        return candidate  // new state back to loop()
    }
}
}

func (raft Raft) leader() {
}

func (raft Raft) candidate() {
	//Self Vote
	votesRxd:=1
    go func(){
    	for _, server := range raft.ClusterConfig.Servers {
        	if server.Id != raft.thisServerId{
        			ServersMap[server.Id].EventCh <- Event{"VoteRequest",Vote{raft.thisServerId, server.Id, false, raft.currentTerm}}
        			fmt.Println("S"+strconv.Itoa(raft.thisServerId)+" Sending vote request to "+strconv.Itoa(server.Id))      		
        	}
	    }
	}()

	    for {
        event := <- raft.EventCh
        switch event.(type) {
        case "ClientAppend":
        	fmt.Println("S"+strconv.Itoa(raft.thisServerId) + ": Rxd Command, But I am a Candidate. NOACT" )
            // Do not handle clients in follower mode. Send it back up the
            // pipe with committed = false
            //ev.logEntry.commited = false
            //commitCh <- ev.logentry
        case "VoteRequest":
        	fmt.Println("S"+strconv.Itoa(raft.thisServerId) + ": Rxd Vote Request from , But I am a CAndi. NOACT" )
            msg = event.payload
            if msg.OrgServerTerm < raft.currentTerm {
            	raft.EventCh = Event{"VoteResponse", Vote{raft.thisServerId, msg.DestServerId, false, raft.currentTerm}}
            }
            else if msg.OrgServerTerm > raft.currentTerm {
            	raft.currentTerm = msg.OrgServerTerm
            }
            //if not already voted in my term
            //    reset timer
             //   reply ok to event.msg.serverid
            //    remember term, leader id (either in log or in separate file)
        case "AppendRPC":
           // reset timer
           // if msg.term < currentterm, ignore
           // reset heartbeat timer
           // upgrade to event.msg.term if necessary
           // if prev entries of my log and event.msg match
           //    add to disk log
            //   flush disk log
            //   respond ok to event.msg.serverid
            else
               respond err.
        case "Timeout": 
        return candidate  // new state back to loop()
    }
}
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
type Raft struct {
	Cluster       *ClusterConfig
	ThisServerId  int
	LeaderId      int
	ServersCount  int
	currentTerm int
	votedFor int
	commitIndex int
	lastApplied int
	EventCh chan Event
}

type LogStruct struct {
	Log_lsn    Lsn
	Log_data   []byte
	Log_commit bool
}

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(clusterConfig *ClusterConfig, thisServerId int, LeaderId int) (*Raft, error) {

	serversCount := 0
	currentTerm = -1
	votedFor = -1
	commitIndex = -1
	lastApplied = -1 
	EventCh := make(chan Event)

	for _, _ = range clusterConfig.Servers {
		serversCount = serversCount + 1
	}

	raft = Raft{clusterConfig, thisServerId, LeaderId,serversCount,currentTerm, votedFor, commitIndex, lastApplied, EventCh}
	fmt.Println("Server "+strconv.Itoa(thisServerId) + " Booted!")
	var err error = nil
	ServersMap[thisServerId] = &raft
	return &raft, err
}

// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "ERR_REDIRECT " + raft.Cluster.Servers[int(e)].Hostname + " " + strconv.Itoa(raft.Cluster.Servers[int(e)].ClientPort) + "\r\n"
}
/*
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
