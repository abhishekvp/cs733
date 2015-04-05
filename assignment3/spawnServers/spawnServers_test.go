package main

import (
	"github.com/abhishekvp/cs733/assignment3/raft"
	"testing"
)

func TestMain(t *testing.T) {

	go main()

	//Test for Leader Elected or not
	for i := 0; i < 5; i++ {
		if raft.ServersMap[i].LeaderId == -1 {
			t.Error("Leader Not Elected!")
		}
	}

	//Test for Safe Leader Election
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if raft.ServersMap[i].LeaderId != raft.ServersMap[j].LeaderId {
				t.Error("Leader Election Unsafe! More than one leader elected!")
			}
		}
	}

}
