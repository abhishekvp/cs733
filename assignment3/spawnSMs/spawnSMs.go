package main

import (
	"github.com/abhishekvp/cs733/assignment3/raft"
	"io/ioutil"
	"encoding/json"
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

	for _, server := range clusterConfig.Servers {
		leaderId:=-1
		_, _ = raft.NewRaft(&clusterConfig, server.Id, leaderId)
	}

}