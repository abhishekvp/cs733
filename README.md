# cs733
Assignment 1. A memcached clone
===============================

This key-value store is loosely based on MemCached. It has been developed using Go.

Testing Instructions
---------------------
* Run the following on your terminal :

  `go get github.com/abhishekvp/cs733/assignment1/`

  `go test github.com/abhishekvp/cs733/assignment1/server`
  
To Do
-----
* Make map concurrency-safe
* Add more automated tests

Protocol Specification
-----------------------
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



  
