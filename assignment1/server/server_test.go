package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

const servAddr string = "localhost:9000"

func Test_Set(t *testing.T) {

	go main()

	//Connect to the Server
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		os.Exit(1)
	}

	// Send set query
	_, err = conn.Write([]byte("set foo 10000 1"))
	if err != nil {
		os.Exit(1)
	}

	time.Sleep(time.Duration(1) * time.Second)

	// Send the value to be set, "bar" in this case
	_, err = conn.Write([]byte("bar"))
	if err != nil {
		os.Exit(1)
	}

	reply := make([]byte, 1024)

	_, err = conn.Read(reply)
	if err != nil {
		os.Exit(1)
	}

	// Assert that the Server responds with "OK"/ Setting works
	if strings.Fields(string(reply))[0] != "OK" {
		t.Error("Setting did not work as expected.")
	}

	time.Sleep(time.Duration(1) * time.Second)

	// Send get query, "foo" as the key
	_, err = conn.Write([]byte("get foo\r\n"))
	if err != nil {
		os.Exit(1)
	}

	time.Sleep(time.Duration(1) * time.Second)

	reply = make([]byte, 1024)

	_, err = conn.Read(reply)
	if err != nil {
		os.Exit(1)
	}
	// Extract the value from the response given by the Server
	fLine := strings.Split(string(reply), "\n")
	val := strings.Fields(string(fLine[2]))[0]
	actual := "bar"

	strSlice1 := fmt.Sprintf("%T", actual)
	strSlice2 := fmt.Sprintf("%T", val)

	// Assert that get value are same/ Getting works
	if strSlice2 != strSlice1 {
		t.Error("Getting did not work as expected.")
	}

	conn.Close()
}
