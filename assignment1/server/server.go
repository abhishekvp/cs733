package main

import (
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
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
var kvMapStruct = struct {
	sync.RWMutex
	kvMap map[string]map[string]string
}{kvMap: make(map[string]map[string]string)}

func main() {

	/**
	TCP Connection Code referred from "Chapter 3. Socket-level Programming, Network programming with Go"
	URL: http://jan.newmarch.name/go/socket/chapter-socket.html
	**/
	tcpAddr, connError := net.ResolveTCPAddr("tcp", port)

	listener, connError := net.ListenTCP("tcp", tcpAddr)

	for {
		conn, genError := listener.Accept()
		checkError(connError, conn)
		checkError(genError, conn)

		// Goroutine to handle multiple clients
		go processClient(conn)

	}

}

func processClient(conn net.Conn) {

	for {

		var clientLine1 [512]byte
		var clientLine2 [512]byte
		var splitClientLine1 []string
		var trimmedClientLine2 string

		// Read First Line from the client
		_, genError := conn.Read(clientLine1[0:])
		checkError(genError, conn)

		//Count the number of spaces in the line received from the client
		count := strings.Count(string(clientLine1[0:]), " ")

		splitClientLine1 = strings.Fields(string(clientLine1[0:]))

		//Read second line, containing the value entered at the client
		if splitClientLine1[0] == "set" || splitClientLine1[0] == "cas" {
			_, genError = conn.Read(clientLine2[0:])
			checkError(genError, conn)

			//Clean the line read, off trailing carriage return and newline
			trimmedClientLine2 = strings.TrimSpace(string(clientLine2[0:]))
			if strings.Contains(trimmedClientLine2, "\r") {
				trimmedClientLine2 = strings.Trim(trimmedClientLine2, "\r")
			}
			if strings.Contains(trimmedClientLine2, "\n") {
				trimmedClientLine2 = strings.Trim(trimmedClientLine2, "\n")
			}

		}

		//Select action based on the first word of the query entered at the client
		switch splitClientLine1[0] {

		case "set":
			//The "Space" count for a legal set query should be minimum 3
			if count >= 3 {

				kvMapStruct.Lock()
				kvMapStruct.kvMap[splitClientLine1[1]] = map[string]string{"exptime": splitClientLine1[2], "numbytes": splitClientLine1[3], "value": trimmedClientLine2, "version": strconv.FormatInt(rand.Int63(), 10)}
				kvMapStruct.Unlock()

				//The Key-Value has pair has 0 expiry time, so it should never expire.
				kvMapStruct.RLock()
				if kvMapStruct.kvMap[splitClientLine1[1]]["exptime"] != "0" {
					kvMapStruct.RUnlock()
					go processExpTime(splitClientLine1[1])
				} else {
					kvMapStruct.RUnlock()
				}
				//Handle the "[noreply]" case
				if count == 4 {
					//Invalid argument instead of "[noreply]"
					if strings.Contains(splitClientLine1[4], "[noreply]") != true {

						kvMapStruct.RUnlock()
						//CommandLine Formatting Error
						ERRCMDERR(conn)
					}
				} else {
					kvMapStruct.RLock()
					_, genError = conn.Write([]byte("OK " + kvMapStruct.kvMap[splitClientLine1[1]]["version"] + "\r\n"))
					kvMapStruct.RUnlock()
				}
			} else {
				kvMapStruct.RUnlock()
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}

		case "get":
			//The "Space" count for a legal get query should be 1
			if count == 1 {

				kvMapStruct.RLock()
				if _, ok := kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["value"]; ok {
					_, genError = conn.Write([]byte("VALUE \r\n" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["numbytes"] + "\r\n" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["value"]))
					kvMapStruct.RUnlock()
				} else {

					kvMapStruct.RUnlock()
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {

				kvMapStruct.RUnlock()
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}

		case "getm":
			//The "Space" count for a legal get query should be 1
			if count == 1 {

				kvMapStruct.RLock()
				if _, ok := kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["value"]; ok {
					_, genError = conn.Write([]byte("VALUE \r\n" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["version"] + "\t" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["exptime"] + "\t" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["numbytes"] + "\r\n" + kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["value"]))
					kvMapStruct.RUnlock()
				} else {

					kvMapStruct.RUnlock()
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {

				kvMapStruct.RUnlock()
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}

		case "cas":
			//The "Space" count for a legal cas query should be 4
			if count >= 4 {

				kvMapStruct.RLock()
				if splitClientLine1[3] == kvMapStruct.kvMap[splitClientLine1[1]]["version"] {

					kvMapStruct.RUnlock()

					kvMapStruct.Lock()
					kvMapStruct.kvMap[splitClientLine1[1]]["value"] = trimmedClientLine2
					kvMapStruct.kvMap[splitClientLine1[1]]["exptime"] = splitClientLine1[2]
					kvMapStruct.kvMap[splitClientLine1[1]]["numbytes"] = splitClientLine1[4]
					kvMapStruct.kvMap[splitClientLine1[1]]["version"] = strconv.FormatInt(rand.Int63(), 10)
					kvMapStruct.Unlock()

					//The Key-Value has pair has 0 expiry time, so it should never expire.
					kvMapStruct.RLock()
					if kvMapStruct.kvMap[splitClientLine1[1]]["exptime"] != "0" {

						kvMapStruct.RUnlock()
						go processExpTime(splitClientLine1[1])
					} else {
						kvMapStruct.RUnlock()
					}
					//Handle the "[noreply]" case
					if count == 5 {
						//Invalid argument instead of "[noreply]"
						if strings.Contains(splitClientLine1[5], "[noreply]") != true {

							kvMapStruct.RUnlock()
							//CommandLine Formatting Error
							ERRCMDERR(conn)
						}
					} else {
						kvMapStruct.RLock()
						_, genError = conn.Write([]byte("OK " + kvMapStruct.kvMap[splitClientLine1[1]]["version"] + "\r\n"))
						kvMapStruct.RUnlock()
						checkError(genError, conn)
					}
				} else {

					kvMapStruct.RUnlock()
					//Version Mismatch. Value not updated.
					ERR_VERSION(conn)
				}
			} else {

				kvMapStruct.RUnlock()
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}

		case "delete":
			//The "Space" count for a legal delete query should be 1
			if count == 1 {
				kvMapStruct.RLock()
				if _, ok := kvMapStruct.kvMap[strings.Trim(splitClientLine1[1], "\r\n")]["value"]; ok {

					kvMapStruct.RUnlock()

					kvMapStruct.Lock()
					delete(kvMapStruct.kvMap, splitClientLine1[1])
					kvMapStruct.Unlock()

					_, genError := conn.Write([]byte("DELETED" + "\r\n"))
					checkError(genError, conn)

				} else {

					kvMapStruct.RUnlock()
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {

				kvMapStruct.RUnlock()
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}
		}
	}

}

/**
* @param key String
* Updates the "exptime" field of the key, by decrementing at the interval of one second.
* Once the exptime is up, the key-value pair is deleted from the Map
*
* Timer and Ticker Code referred from Go By Example
* URL: https://gobyexample.com/tickers
 */
func processExpTime(key string) {

	ticker := time.NewTicker(time.Millisecond * 1000)
	var t time.Time
	kvMapStruct.RLock()
	exptime, _ := strconv.Atoi(kvMapStruct.kvMap[key]["exptime"])
	kvMapStruct.RUnlock()
	go func() {
		for t = range ticker.C {
			exptime = exptime - 1
			// Check for case when the key value pair is deleted or not found
			kvMapStruct.RLock()
			if _, ok := kvMapStruct.kvMap[key]["exptime"]; ok {
				kvMapStruct.RUnlock()

				kvMapStruct.Lock()
				kvMapStruct.kvMap[key]["exptime"] = strconv.Itoa(exptime)
				kvMapStruct.Unlock()
			} else {
				kvMapStruct.RUnlock()
			}

		}
	}()
	time.Sleep(time.Duration(exptime) * time.Second)
	ticker.Stop()

	kvMapStruct.Lock()
	delete(kvMapStruct.kvMap, key)
	kvMapStruct.Unlock()

}

//Error handler
func checkError(genError error, conn net.Conn) {
	if genError != nil {
		err := "ERR_INTERNAL\r\n"
		_, genError = conn.Write([]byte(err))
	}
}

func ERRCMDERR(conn net.Conn) {
	var genError error
	err := "ERRCMDERR\r\n"
	_, genError = conn.Write([]byte(err))
}

func ERRNOTFOUND(conn net.Conn) {
	var genError error
	err := "ERRNOTFOUND\r\n"
	_, genError = conn.Write([]byte(err))
}

func ERR_VERSION(conn net.Conn) {
	var genError error
	err := "ERR_VERSION\r\n"
	_, genError = conn.Write([]byte(err))
}
