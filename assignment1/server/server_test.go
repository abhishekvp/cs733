package main

import (
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const servAddr string = "localhost:9000"

var conn net.Conn

func TestMain(t *testing.T) {

	go main()

	//Test server for a single client
	clientTest(t)

	//Test server for 1000 clients
	for i:=0; i<1000; i++{

		go func(){
			clientTest(t)
		}()

	}
}

	

func clientTest(t *testing.T) {	
	
	//Connect to the Server
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		os.Exit(1)
	}

	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		os.Exit(1)
	}

	ch := make(chan string)
	// Start a goroutine to read from our net connection
	go func(ch chan string) {
		for {

			data := make([]byte, 1024)
			// try to read the data
			size, err := conn.Read(data)

			if err != nil {
				t.Error(err)
			}

			// send data if we read some.
			data = data[:size]
			ch <- string(data)

		}
	}(ch)

	//#1 TestCase SET
	serverCmd("set foo 0 1\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("bar")
	serverReply := <-ch
	assert("OK", strings.Fields(string(serverReply))[0], t)

	//To be used for next test - CAS
	version := strings.Fields(string(serverReply))[1]

	//#2 TestCase CAS
	serverCmd("cas foo 0 " + version + " 2\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("changedBar")
	serverReply = <-ch
	assert("OK", strings.Fields(string(serverReply))[0], t)

	//#3 TestCase GET
	serverCmd("get foo\n")
	serverReply = <-ch
	assert("changedBar", strings.Fields(string(serverReply))[2], t)

	//#4 TestCase GETM
	serverCmd("getm foo\n")
	serverReply = <-ch
	assert("VALUE \r\n([0-9]*)"+"\t"+"0"+"\t"+"2"+"\r\n"+"changedBar", string(serverReply), t)

	//#5 TestCase DELETE
	serverCmd("delete foo\n")
	serverReply = <-ch
	assert("DELETED", strings.Fields(string(serverReply))[0], t)

	//#6 TestCase CAS Version Error
	serverCmd("cas foo 0 1000 2\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("changedBar")
	serverReply = <-ch
	assert("ERR_VERSION", strings.Fields(string(serverReply))[0], t)

	//#7 TestCase SET Error
	serverCmd("set inv 0\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("test")
	serverReply = <-ch
	assert("ERRCMDERR", strings.Fields(string(serverReply))[0], t)

	//#8 TestCase GET Key Not Found Error
	serverCmd("get foo\n")
	serverReply = <-ch
	assert("ERRNOTFOUND", strings.Fields(string(serverReply))[0], t)

	//#9 TestCase Delete Key Not Found Error
	serverCmd("delete foo\n")
	serverReply = <-ch
	assert("ERRNOTFOUND", strings.Fields(string(serverReply))[0], t)

	//#10 TestCase GETM Key Not Found Error
	serverCmd("getm bar\n")
	serverReply = <-ch
	assert("ERRNOTFOUND", strings.Fields(string(serverReply))[0], t)

	//#11 TestCase GET Error
	serverCmd("get\n")
	serverReply = <-ch
	assert("ERRCMDERR", strings.Fields(string(serverReply))[0], t)

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
