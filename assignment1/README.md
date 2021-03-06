Assignment 1. A Memcached Clone
===============================

This key-value store is a client-server architecture based network application. It is loosely based on Memcached. TCP is used for communication. It has been developed using Go. 

This is the first assignment of the course CS733 - Engineering a Cloud at IIT Bombay.

### Testing Instructions

* Run the following on your terminal :

  `go get github.com/abhishekvp/cs733/assignment1/`

  `go test github.com/abhishekvp/cs733/assignment1/server`
  
### Included Test Cases

* Set a key-value.
* Set - Invalid Commandline Formatting
* Get a value.
* Get - Invalid Commandline Formatting
* Get - Invalid key request
* Get Metadata for a key-value
* Get Metadata - Invalid key request
* Compare and Swap a key-value
* Compare and Swap - Invalid Version Error
* Delete a key-value
* Delete - Invalid Key request
* Concurrency Test(all above tests) for 100 clients

Time required for tests to complete: 4 - 5 seconds

### Usage

* Change directory to cs733/assignment1/server and run the following on the terminal to start the server and listen on port 9000:

  `go run server.go`

* Keeping the terminal used above open, open another instance of terminal and run the following to connect to the server:

  `telnet <Server IP Address><"localhost" for testing locally> 9000`

Once connected, the client can issue commands to server namely - SET, GET, GETM, CAS, DELETE, as per protocol specification

### Mechanism

* The server listens on port 9000 for client commands. The client can connect to the server using telnet.
* Once connected, the client can issue SET, GET, GETM, CAS, DELETE commands to the server.
* The server stores the key-value in a map of maps.

```
  /*
  * Map of Maps to store key-values.
  * kvMap[key]["exptime"]
  * kvMap[key]["value"]
  * kvMap[key]["numbytes"]
  * kvMap[key]["version"]
  */
```
* The expiry is handled by making use of goroutines. Once a key-value is set or cas-ed, a goroutine is invoked to update the expiry time in the Map and delete the key-value once the expiry time is up.
* The Map is made concurrency safe using Mutex - `sync.RWMutex` from the `sync` package in Go.


### Protocol Specification

* Set: create the key-value pair, or update the value if it already exists.

  `set <key> <exptime> <numbytes> [noreply]\r\n`
  
  `<value bytes>\r\n`

  The server responds with:

  `OK <version>\r\n`

  where version is a unique 64-bit number (in decimal format) assosciated with the key.

* Get: Given a key, retrieve the corresponding key-value pair

  `get <key>\r\n`

  The server responds with the following format (or one of the errors described later)

  `VALUE <numbytes>\r\n`
  
  `<value bytes>\r\n`

* Get Meta: Retrieve value, version number and expiry time left

  `getm <key>\r\n`

  The server responds with the following format (or one of the errors described below)

  `VALUE <version> <exptime> <numbytes>\r\n`
  
  `<value bytes>\r\n`

* Compare and swap. This replaces the old value (corresponding to key) with the new value only if the version is still the same.

  `cas <key> <exptime> <version> <numbytes> [noreply]\r\n`
  
  `<value bytes>\r\n`

  The server responds with the new version if successful (or one of the errors described late)

  `OK <version>\r\n`

* Delete key-value pair

  `delete <key>\r\n`

  Server response (if successful)

  `DELETED\r\n`
  
#### Errors Returned
* `ERRCMDERR` - Commandline Formatting Error
* `ERRNOTFOUND` - Key Not Found Error
* `ERR_VERSION` - Version Mismatch Error
* `ERR_INTERNAL` - Internal Error



  
