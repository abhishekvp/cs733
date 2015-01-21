package main
import (
"net"
"os"
"testing"
"strings"
"time"
)
 
const servAddr string = "localhost:9000"

func Test_Set(t *testing.T) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
	os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
	os.Exit(1)
	}

	_, err = conn.Write([]byte("set foo 0 1"))
	if err != nil {
	os.Exit(1)
	}

	time.Sleep(time.Duration(1)*time.Second)

	_, err = conn.Write([]byte("bar"))
	if err != nil {
	os.Exit(1)
	}

	reply := make([]byte, 1024)
	 
	_, err = conn.Read(reply)
	if err != nil {
	os.Exit(1)
	}

	if strings.Fields(string(reply))[0] != "OK" {
		t.Error("Setting did not work as expected.")
		
	}
	conn.Close()

}
