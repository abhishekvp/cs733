package main

import (
	"encoding/json"
	"github.com/abhishekvp/cs733/assignment3/raft"
	"io/ioutil"
	"sync"
	"time"
	"log"
	//"runtime"
)

func main() {
	var clusterConfig raft.ClusterConfig
	serverConfig, err := ioutil.ReadFile("/home/avp/GO/src/github.com/abhishekvp/cs733/assignment3/servers.json")
	if err != nil {
		panic(err)
	}

	err_json := json.Unmarshal(serverConfig, &clusterConfig)
	if err_json != nil {
		panic(err_json)
	}

	//var raftInstance raft.Raft

	var wg sync.WaitGroup

	for _, server := range clusterConfig.Servers {
		leaderId := -1
		var raftInstance, _ = raft.NewRaft(&clusterConfig, server.Id, leaderId)
		wg.Add(1)
		go raftInstance.Loop(wg)
	}
	log.Println("Sleeping for 2 Seconds")
	time.Sleep(time.Duration(2000) * time.Millisecond)
	log.Println("Woke Up after 2 Seconds")
	raft.MapStruct.RLock()
	leader := raft.MapStruct.ServersMap[0].LeaderId
	log.Println("Obtained Leader! Append Called")
	_,_ = raft.MapStruct.ServersMap[leader].Append("Hello World")
	wg.Wait()
	//raftInstance.PrintAllRafts()

}
