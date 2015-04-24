package main

import (
	"encoding/json"
	"github.com/abhishekvp/cs733/assignment4/raft"
	"io/ioutil"
	"sync"
	"strconv"
	"os"
	//"time"
	//"log"
	//"runtime"
)

//This code starts each Server - the Raft instance initialisation, the KV Store , and the ConnHandler
func main() {

    //Get Server ID from Command Line
    serverId, err := strconv.Atoi(os.Args[1])

	var clusterConfig raft.ClusterConfig
	serverConfig, err := ioutil.ReadFile("/home/avp/GO/src/github.com/abhishekvp/cs733/assignment4/servers.json")
	if err != nil {
		panic(err)
	}

	err_json := json.Unmarshal(serverConfig, &clusterConfig)
	if err_json != nil {
		panic(err_json)
	}

	//var raftInstance raft.Raft

	var wg sync.WaitGroup

	//for _, server := range clusterConfig.Servers {
		leaderId := -1

		commitCh := make(chan raft.LogStruct)
		var raftInstance, _ = raft.NewRaft(&clusterConfig, serverId, leaderId, commitCh)
		wg.Add(1)
		go raftInstance.Loop(wg)
	//}
	/*
	log.Println("Sleeping for 2 Seconds")
	time.Sleep(time.Duration(2000) * time.Millisecond)
	log.Println("Woke Up after 2 Seconds")
	//raft.MapStruct.RLock()
	leader := raft.ServersMap[0].LeaderId
	log.Println("Obtained Leader! Append Called")
	_,_ = raft.ServersMap[leader].Append("Hello World")
	time.Sleep(time.Duration(2000) * time.Millisecond)
	log.Println("Woke Up after 2 Seconds")
	//raft.MapStruct.RLock()
	log.Println("Obtained Leader! Append Called")
	_,_ = raft.ServersMap[leader].Append("Thank You All !")
		time.Sleep(time.Duration(2000) * time.Millisecond)
	log.Println("Woke Up after 2 Seconds")
	//raft.MapStruct.RLock()
	log.Println("Obtained Leader! Append Called")
	_,_ = raft.ServersMap[leader].Append("Thats Great! It Worked")
	
	//raftInstance.PrintAllRafts()
	*/
	wg.Wait()
}
