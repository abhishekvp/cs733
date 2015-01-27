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

	//#2 TestCase GET
	serverCmd("get foo\n")
	serverReply = <-ch
	assert("bar", strings.Fields(string(serverReply))[2], t)

	//#3 TestCase GETM
	serverCmd("getm foo\n")
	serverReply = <-ch
	assert("VALUE \r\n([0-9]*)"+"\t"+"0"+"\t"+"1"+"\r\n"+"bar", string(serverReply), t)

	//#4 TestCase SET Error
	serverCmd("set inv 0\n")
	time.Sleep(time.Duration(1) * time.Second)
	serverCmd("test")
	serverReply = <-ch
	assert("ERRCMDERR", strings.Fields(string(serverReply))[0], t)

	//#5 TestCase GET Key Not Found Error
	serverCmd("get bar\n")
	serverReply = <-ch
	assert("ERRNOTFOUND", strings.Fields(string(serverReply))[0], t)

	//#6 TestCase GETM Key Not Found Error
	serverCmd("getm bar\n")
	serverReply = <-ch
	assert("ERRNOTFOUND", strings.Fields(string(serverReply))[0], t)

	//#7 TestCase GET Error
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
