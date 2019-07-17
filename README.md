## Server
Need to pass -psqlurl arg to specify a postgre server

Server listens and accept connections.
After succesful login, it start to send hardcoded jobs to connected peer.
```bash
Example server usage:
go run server.go -host localhost -port 3030 -psqlurl "postgres://postgres:123321@localhost:5432/picus?sslmode=disable"
```

## Client
After connecting makes a login request, if accepted it listens for jobs
If Job message received executes and write result back to connection

Example client usage:
```bash
go run client.go -peer 1 -host localhost -token someToken
```

