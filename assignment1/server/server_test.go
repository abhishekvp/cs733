package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const servAddr string = "localhost:9000"

var conn net.Conn

func Test_Cases(t *testing.T) {
	
	go main()

	//Connect to the Server

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		os.Exit(1)
	}

	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		os.Exit(1)
	}

	ch := make(chan []byte)
	// Start a goroutine to read from our net connection
	go func(ch chan []byte) {
		for {
			// try to read the data
			data := make([]byte, 512)
			_, err := conn.Read(data)
			if err != nil {
				os.Exit(1)
			}
			// send data if we read some.
			ch <- data
		}
	}(ch)

	//#1 TestCase SET
	serverCmd("set foo 10000 1\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("bar")
	serverReply := make([]byte, 512)
	serverReply = <-ch
	assert("OK", strings.Fields(string(serverReply))[0], t)

	time.Sleep(time.Duration(1) * time.Second)

	//#2 TestCase GET
	serverCmd("get foo\n")
	serverReply = make([]byte, 512)
	serverReply = <-ch
	assert("bar", strings.Fields(string(serverReply))[2], t)
	time.Sleep(time.Duration(1) * time.Second)

	conn.Close()

}

func serverCmd(cmd string) {

	_, err := conn.Write([]byte(cmd))
	if err != nil {
		os.Exit(1)
	}

}
func assert(expectedResponse string, serverResponse string, t *testing.T) {

	equal, _ := regexp.MatchString(expectedResponse, serverResponse)
	if !equal {
		t.Error("Expected " + expectedResponse + " , Got " + serverResponse + " instead.")
	}

}
