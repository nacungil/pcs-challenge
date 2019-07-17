//takes peerId, token... to connect. Check flags
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

var host = flag.String("host", "localhost", "The hostname or IP to connect to; defaults to \"localhost\".")
var port = flag.Int("port", 3030, "The port to connect to; defaults to 3030.")
var peerID = flag.Int("peer", 0, "Specifies peer id to connect to server")
var token = flag.String("token", "", "Specifies connection token")
var done = make(chan interface{}, 1)
var wg sync.WaitGroup

func main() {
	flag.Parse()
	dest := *host + ":" + strconv.Itoa(*port)
	fmt.Printf("Connecting to %s...\n", dest)

	conn, err := net.Dial("tcp", dest)
	// jobCh := make(chan []byte)
	if err != nil {
		if _, t := err.(*net.OpError); t {
			fmt.Println("Some problem connecting.")
		} else {
			fmt.Println("Unknown error: " + err.Error())
		}
		os.Exit(1)
	}
	//After connecting
	//send register message
	wg.Add(1)
	go loginAndreceiveJobs(conn)
	wg.Wait()
	if err := conn.Close(); err == nil {
		fmt.Println("Server Dropped, closed connection")
	}
}

func loginAndreceiveJobs(conn net.Conn) {
	log.Printf("Logging in peer_id: %d, token: %s\n", *peerID, *token)

	req := fmt.Sprintf(`{"msg":"login", "peer_id": "%d", `+
		`"token": "%s", "server_ip":"%s:%d\n"}`, *peerID, *token, *host, *port)
	conn.Write([]byte(req + "\n"))

	scanner := bufio.NewScanner(conn)
	for {
		//wait for login response
		ok := scanner.Scan()
		if !ok {
			break
		}
		b := scanner.Bytes()
		log.Printf("Login response %s\n", b)

		var v map[string]interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			log.Println(err)
			return
		}
		if v["status"] == "rejected" {
			return
		}
		if v["status"] == "accepted" {
			break
		}
	}
	receiveJobs(conn)
	wg.Done()
}

func listenJobChannel(conn net.Conn, c chan string) {
	for j := range c {
		if _, err := conn.Write([]byte(j + "\n")); err != nil {
			fmt.Println(err)
		}
	}
}

func receiveJobs(conn net.Conn) {
	scanner := bufio.NewScanner(conn)

	resCh := make(chan string)
	defer close(resCh)
	go listenJobChannel(conn, resCh)

	for {
		if ok := scanner.Scan(); !ok {
			break
		}
		b := scanner.Bytes()
		fmt.Printf("received job %s\n", b)
		var data map[string]interface{}
		if err := json.Unmarshal(b, &data); err != nil {
			log.Println(err)
			break
		}
		switch data["msg"] {
		case "jobs":
			job, ok := data["job"].(map[string]interface{})
			if !ok {
				break
			}
			j, ok := job["command"].(string)

			ok, res := doJob(j)
			if !ok {
				fmt.Println("Problem executing job")
			}
			resCh <- string(res)
		}
	}
}

func doJob(comm string) (bool, []byte) {
	var res = make(map[string]string, 100)
	res["msg"] = "result"
	res["peer_id"] = strconv.Itoa(*peerID)
	res["token"] = *token
	switch comm {
	case "free":
		res["result"] = "file1 file2 file11 file12"

	case "ls":
		res["result"] = "total:7.4G used: 514M free:1.6G"
	default:
		res["result"] = "Unknown command"
	}
	b, err := json.Marshal(res)
	if err != nil {
		return false, nil
	}
	return true, b
}

type errorString struct {
	s string
}

func (e errorString) Error() string {
	return e.s
}
