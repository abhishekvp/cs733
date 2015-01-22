package main

import (
	"fmt"
	"math/rand"
	"net"
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

	var bufLine1 [512]byte
	var bufLine2 [512]byte
	var cLine1 []string
	var cLine2 string


		// Read First Line from the client
		_, genError := conn.Read(bufLine1[0:])
		checkError(genError, conn)
		
		//Count the number of spaces in the line received from the client
		count:=strings.Count(string(bufLine1[0:])," ")		
		fmt.Println("Count = "+ strconv.Itoa(count))
		
		cLine1 = strings.Fields(string(bufLine1[0:]))

		//Read second line, containing the value entered at the client
		if cLine1[0] == "set" || cLine1[0] == "cas" {
			_, genError = conn.Read(bufLine2[0:])
			checkError(genError, conn)

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
			//The "Space" count for a legal set query should be minimum 3
			if count >= 3 {
				kvMap[cLine1[1]] = map[string]string{"exptime": cLine1[2], "numbytes": cLine1[3], "value": cLine2, "version": strconv.FormatInt(rand.Int63(), 10)}
				//The Key-Value has pair has 0 expiry time, so it should never expire.
				if kvMap[cLine1[1]]["exptime"] != "0" {
					go processExpTime(cLine1[1])
				}
				//Handle the "[noreply]" case
				if count == 4 {
					//Invalid argument instead of "[noreply]"
					if strings.Contains(cLine1[4],"[noreply]") != true {
						//CommandLine Formatting Error
						ERRCMDERR(conn)
					}
				} else {
					_, genError = conn.Write([]byte("OK " + kvMap[cLine1[1]]["version"] + "\n"))
				}
			} else {
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}

		case "get":
			//The "Space" count for a legal get query should be 1
			if count == 1 {
				if _, ok := kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]; ok {
					_, genError = conn.Write([]byte("VALUE "))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["numbytes"] + "\n"))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]))
				} else {
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {
				//CommandLine Formatting Error
				ERRCMDERR(conn)	
			}

		case "getm":
			//The "Space" count for a legal get query should be 1
			if count == 1 {
				if _, ok := kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]; ok {
					_, genError = conn.Write([]byte("VALUE "))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["version"] + "\t"))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["exptime"] + "\t"))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["numbytes"] + "\n"))
					_, genError = conn.Write([]byte(kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]))
				} else {
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {
				//CommandLine Formatting Error
				ERRCMDERR(conn)	
			}

		case "cas":
			//The "Space" count for a legal cas query should be 4
			if count >= 4 {

				if cLine1[3] == kvMap[cLine1[1]]["version"] {

					kvMap[cLine1[1]]["value"] = cLine2
					kvMap[cLine1[1]]["exptime"] = cLine1[2]
					kvMap[cLine1[1]]["numbytes"] = cLine1[4]
					kvMap[cLine1[1]]["version"] = strconv.FormatInt(rand.Int63(), 10)
					//The Key-Value has pair has 0 expiry time, so it should never expire.
					if kvMap[cLine1[1]]["exptime"] != "0" {
						go processExpTime(cLine1[1])
					}
					//Handle the "[noreply]" case
					if count == 5 {
						//Invalid argument instead of "[noreply]"
						if strings.Contains(cLine1[5],"[noreply]") != true {
							//CommandLine Formatting Error
							ERRCMDERR(conn)
						}
					} else {
						_, genError = conn.Write([]byte("OK " + kvMap[cLine1[1]]["version"] + "\n"))
						checkError(genError, conn)
					}
				} else {
				//Version Mismatch. Value not updated.
				ERR_VERSION(conn)
				}
			} else {
				//CommandLine Formatting Error
				ERRCMDERR(conn)
			}


		case "delete":
			//The "Space" count for a legal delete query should be 1
			if count == 1 {
				if _, ok := kvMap[strings.Trim(cLine1[1], "\r\n")]["value"]; ok {
					delete(kvMap, cLine1[1])
					_, genError := conn.Write([]byte("DELETED"+"\n"))
					checkError(genError, conn)

				} else {
					//Key Not Found Error
					ERRNOTFOUND(conn)
				}
			} else {
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
 */
func processExpTime(key string) {



		ticker := time.NewTicker(time.Millisecond * 1000)
		var t time.Time
		exptime, _ := strconv.Atoi(kvMap[key]["exptime"])
		go func() {
			for t = range ticker.C {
				exptime = exptime - 1
				// Check for case when the key value pair is deleted or not found
				if _, ok := kvMap[key]["exptime"]; ok {
					kvMap[key]["exptime"] = strconv.Itoa(exptime)
				}

			}
		}()
		time.Sleep(time.Duration(exptime) * time.Second)
		ticker.Stop()
		delete(kvMap, key)
	

}

//Error handler
func checkError(genError error, conn net.Conn) {
	if genError != nil {
		err := "ERR_INTERNAL\n"
		_, genError = conn.Write([]byte(err))
	}
}

func ERRCMDERR(conn net.Conn) {
	var genError error
	err := "ERRCMDERR\n"
	_, genError = conn.Write([]byte(err))
}

func ERRNOTFOUND(conn net.Conn) {
	var genError error
	err := "ERRNOTFOUND\n"
	_, genError = conn.Write([]byte(err))
}


func ERR_VERSION(conn net.Conn) {
	var genError error
	err := "ERR_VERSION\n"
_	, genError = conn.Write([]byte(err))
}
