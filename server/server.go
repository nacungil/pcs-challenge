package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
)

var (
	psqlURL      = flag.String("psqlurl", "postgres://postgres:123321@localhost:5432/pc2?sslmode=disable", "Postgres server address")
	peers        = make(map[int]Peer, 10)
	wg           sync.WaitGroup
	ln           net.Listener
	addr         = flag.String("addr", "", "The address to listen to; default is \"\" (all interfaces).")
	port         = flag.Int("port", 3030, "The port to listen on; default is 3030.")
	migrationDir = flag.String("migration-dir", "../migrations", "Directory where the migration files are located ?")
	db           *sql.DB
	quit         = make(chan os.Signal, 1)
)

type Peer struct {
	id    int
	token string
}

func init() {
	dt, err := sql.Open("postgres", *psqlURL)
	if err != nil {
		log.Fatal(err)
	}
	db = dt
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatal(err)
	}

	// Migrate all the way up ...
	if err := m.Up(); err != nil {
		log.Println("error on upping migration", err)
	}
	if err = db.Ping(); err != nil {
		log.Println(err)
	}
}

type StudentD struct {
	ID   string `db:"uuid"`
	Name string `db:"name"`
	Data string `db:"data"`
}

func main() {
	peers[0] = Peer{0, "token0"}
	peers[1] = Peer{1, "token1"}

	// connCh := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Server starting")
	wg.Add(1)
	go startServer()

	//handle shutdown
	<-quit
	db.Close()
	wg.Done()
	stopServer()
	fmt.Println("Server Stopped")
}

func startServer() {
	src := *addr + ":" + strconv.Itoa(*port)
	ln, err := net.Listen("tcp", src)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("Listening on %s.\n", src)
	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error on accepting connection: %s\n", err)
			break
		}

		// connChan <- conn
		wg.Add(1)
		go handleConnection(conn)
	}
	wg.Done()
}

func stopServer() {
	//TODO need to check closing active connections properly
	fmt.Println("Stopping Server")
	// ln := <-lnChan
	if ln != nil {
		ln.Close()
	}
	fmt.Println("Closed listener")
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	fmt.Println("Client connected from " + remoteAddr)
	//TODO write log to db
	scanner := bufio.NewScanner(conn)

	for {
		ok := scanner.Scan()
		if !ok {
			break
		}
		b := scanner.Bytes()

		handleMessage(b, conn)
	}
	//TODO write log to db
	fmt.Println("Client at " + remoteAddr + " disconnected.")
	wg.Done()
}

func handleMessage(message []byte, conn net.Conn) {
	// fmt.Printf("Handling message: %s\n", message)
	jobs := []string{
		`{ "msg": "jobs", "job": { "name":"List Files", "command": "ls" } }`,
		`{ "msg": "jobs", "job": {  "name":"Free Memory Space", "command": "free" } }`,
		`{ "msg": "jobs", "job": { "name":"List Files", "command": "ls" } }`,
		`{ "msg": "jobs", "job": {  "name":"Free Memory Space", "command": "free" } }`,
		`{ "msg": "jobs", "job": { "name":"List Files", "command": "ls" } }`,
		`{ "msg": "jobs", "job": {  "name":"Free Memory Space", "command": "free" } }`,
	}
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Println(err)
		return
	}

	switch data["msg"] {
	case "login":
		ok, err := handleLogin(conn, data)
		if err != nil {
			fmt.Println(err)
		}
		if ok {
			//logged to db
			_, err := db.Exec(`INSERT INTO logs (peer_id,action,created_on) VALUES ($1,$2,$3)`,
				data["peer_id"],
				"login",
				time.Now(),
			)
			if err != nil {
				log.Println(err)
			}
			//Send jobs after succesfull login
			for _, j := range jobs {
				fmt.Printf("sending job : %s\n", j)
				if _, err := conn.Write([]byte(j + "\n")); err != nil {
					fmt.Println(err)
				}
			}
		}
	case "result":
		fmt.Printf("Result: %s\n", message)
		_, err := db.Exec(`INSERT INTO results (peer_id,res,created_on) VALUES ($1,$2,$3)`,
			data["peer_id"],
			data["result"],
			time.Now(),
		)
		if err != nil {
			log.Println(err)
		}
	default:
		conn.Write([]byte("Unknown message"))
	}
}

func handleLogin(conn net.Conn, data map[string]interface{}) (bool, error) {
	fmt.Println("Checking user ")
	//TODO check user from database / map
	// if pid , ok := data["peer_id"].(int); !ok {
	// 	false,
	// }
	// if peers[]
	v := make(map[string]interface{})

	//if fail handle

	//if success

	v["msg"] = "register"
	v["status"] = "accepted"
	if s, ok := data["peer_id"].(string); ok {
		v["peer_id"] = s
	}

	res, err := json.Marshal(v)
	if err != nil {
		return false, err
	}
	r := fmt.Sprintf("%s%s", res, "\n")
	if _, err := conn.Write([]byte(r)); err != nil {
		fmt.Printf("Error sending Message:\n%s\n", r)
		return false, err
	}
	fmt.Printf("Sent to Client: %s", r)
	return true, nil
}
