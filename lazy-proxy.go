package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var (
	idleTimeout    time.Duration
	listenPort     string
	forwardAddress string
	commandArgs    []string
	verbose        bool

	startProcessChan = make(chan struct{}, 1)
	activityPingChan = make(chan struct{}, 1)
	quitChan         = make(chan struct{})

	lastActivity time.Time
	processLock  sync.Mutex
	processCmd   *exec.Cmd
)

func main() {
	var idleStr, listenStr, forwardStr string

	flag.StringVar(&idleStr, "idle", "", "Idle timeout duration")
	flag.StringVar(&listenStr, "listen", "", "Port to listen on")
	flag.StringVar(&forwardStr, "forward", "", "Address to forward to")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.Parse()

	if idleStr == "" || listenStr == "" || forwardStr == "" {
		log.Fatalf("[proxy] Missing required flags: -idle, -listen, -forward")
	}

	commandArgs = flag.Args()
	if len(commandArgs) == 0 {
		log.Fatalf("[proxy] No command provided to run")
	}

	var err error
	idleTimeout, err = time.ParseDuration(idleStr)
	if err != nil {
		log.Fatalf("[proxy] Invalid idle duration: %v", err)
	}

	listenPort = listenStr
	forwardAddress = forwardStr

	lastActivity = time.Now()
	go processManager()

	log.Println("[proxy] Starting lazy proxy on", listenPort)
	ln, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Fatalf("[proxy] Failed to listen on %s: %v", listenPort, err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[proxy] Failed to accept connection: %v", err)
			continue
		}
		if verbose {
			log.Printf("[proxy] New connection from %s", conn.RemoteAddr())
		}
		go handleConnection(conn)
	}
}

func processManager() {
	for {
		select {
		case <-startProcessChan:
			processLock.Lock()
			if processCmd == nil {
				cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				if err := cmd.Start(); err != nil {
					log.Fatalf("[proxy] Failed to start backend: %v", err)
				}
				processCmd = cmd
				if verbose {
					log.Println("[proxy] Backend process started")
				}
				go func() {
					err := cmd.Wait()
					log.Printf("[proxy] Backend exited: %v", err)
					os.Exit(1)
				}()
			}
			processLock.Unlock()
		case <-activityPingChan:
			lastActivity = time.Now()
		case <-time.After(5 * time.Second): // check every 5 seconds
			processLock.Lock()
			if processCmd != nil && time.Since(lastActivity) > idleTimeout {
				if verbose {
					log.Println("[proxy] Idle timeout reached, killing backend")
				}
				_ = syscall.Kill(-processCmd.Process.Pid, syscall.SIGTERM)
				processCmd = nil
			}
			processLock.Unlock()
		case <-quitChan:
			return
		}
	}
}

func handleConnection(client net.Conn) {
	defer client.Close()
	select {
	case startProcessChan <- struct{}{}:
	default:
	}

	var server net.Conn
	for {
		var err error
		server, err = net.Dial("tcp", forwardAddress)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer server.Close()

	activityPingChan <- struct{}{}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(server, client)
		activityPingChan <- struct{}{}
	}()
	go func() {
		defer wg.Done()
		io.Copy(client, server)
		activityPingChan <- struct{}{}
	}()
	wg.Wait()
}

func isConnClosed(conn net.Conn) bool {
	one := []byte{}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	_, err := conn.Read(one)
	if os.IsTimeout(err) {
		_ = conn.SetReadDeadline(time.Time{})
		return false
	}
	return true
}
