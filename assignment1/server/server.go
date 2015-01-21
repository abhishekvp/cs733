package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const port string = ":9000"

/*
* Map of Maps to store data.
* 	kvMap[key]["exptime"]
* 	kvMap[key]["value"]
* 	kvMap[key]["numbytes"]
* 	kvMap[key]["version"]
 */
var kvMap map[string]map[string]string

func main() {

	kvMap = make(map[string]map[string]string)

	/**
	TCP Connection Code referred from "Chapter 3. Socket-level Programming, Network programming with Go"
	URL: http://jan.newmarch.name/go/socket/chapter-socket.html
	**/
	tcpAddr, err := net.ResolveTCPAddr("tcp", port)
	checkError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Goroutine to handle multiple clients
		go processClient(conn)

	}
}

func processClient(conn net.Conn) {

	var bufLine1 [512]byte
	var bufLine2 [512]byte
	var cLine1 []string
	var cLine2 string
	var err2 error

	for {
		// Read First Line from the client
		_, err := conn.Read(bufLine1[0:])
		if err != nil {
			checkError(err)
			return
		}

		cLine1 = strings.Fields(string(bufLine1[0:]))

		//Read second line, containing the value entered at the client
		if cLine1[0] == "set" || cLine1[0] == "cas" {
			_, err = conn.Read(bufLine2[0:])
			if err != nil {
				checkError(err)
				return
			}
			//Clean the line read, off trailing carriage return and newline
			cLine2 = strings.TrimSpace(string(bufLine2[0:]))
			if strings.Contains(cLine2, "\r") {
				cLine2 = strings.Trim(cLine2, "\r")
			}
			if strings.Contains(cLine2, "\n") {
				cLine2 = strings.Trim(cLine2, "\n")
			}

		}

		//Select action based on the first word of the query entered at the client
		switch cLine1[0] {

		case "set":
			kvMap[cLine1[1]] = map[string]string{"exptime": cLine1[2], "numbytes": cLine1[3], "value": cLine2, "version": strconv.FormatInt(rand.Int63(), 10)}
			//The Key-Value has pair has 0 expiry time, so it should never expire.
			if kvMap[cLine1[1]]["exptime"] != "0" {
				go processExpTime(cLine1[1])
			}
			//Check for [noreply]
			//if strings.Contains(cLine1[4],"[noreply]") != true {
			_, err2 = conn.Write([]byte("OK " + kvMap[cLine1[1]]["version"] + "\n"))
			//}

		case "get":
			_, err2 = conn.Write([]byte("VALUE "))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["numbytes"] + "\n"))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]))

		case "getm":
			_, err2 = conn.Write([]byte("VALUE "))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["version"] + "\t"))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["exptime"] + "\t"))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["numbytes"] + "\n"))
			_, err2 = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]))

		case "cas":
			if cLine1[3] == kvMap[cLine1[1]]["version"] {

				kvMap[cLine1[1]]["value"] = cLine2
				kvMap[cLine1[1]]["exptime"] = cLine1[2]
				kvMap[cLine1[1]]["numbytes"] = cLine1[4]
				kvMap[cLine1[1]]["version"] = strconv.FormatInt(rand.Int63(), 10)
				//The Key-Value has pair has 0 expiry time, so it should never expire.
				if kvMap[cLine1[1]]["exptime"] != "0" {
					go processExpTime(cLine1[1])
				}
				//Check for [noreply]
				//if strings.Contains(cLine1[5],"[noreply]") != true {
				_, err2 = conn.Write([]byte("OK " + kvMap[cLine1[1]]["version"] + "\n"))
				//}
			}

		case "delete":
			delete(kvMap, cLine1[1])
			_, err2 = conn.Write([]byte("DELETED\n"))

		}

		if err2 != nil {
			return
		}

	}

}

/**
* @param key String
* Updates the "exptime" field of the key, by decrementing at the interval of one second.
* Once the exptime is up, the key-value pair is deleted from the Map
 */
func processExpTime(key string) {

	ticker := time.NewTicker(time.Millisecond * 1000)
	var t time.Time
	exptime, err := strconv.Atoi(kvMap[key]["exptime"])

	if err != nil {
		fmt.Println("Error")
		return
	}

	go func() {
		for t = range ticker.C {
			exptime = exptime - 1
			kvMap[key]["exptime"] = strconv.Itoa(exptime)

		}
	}()
	time.Sleep(time.Duration(exptime) * time.Second)
	ticker.Stop()
	delete(kvMap, key)

}

//Error handler
func checkError(err error) {
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
	}
}
