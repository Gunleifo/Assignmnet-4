package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	ring "github.com/Gunleifo/Assignmnet-4/proto"
	"google.golang.org/grpc"
)

type peer struct {
	ring.UnimplementedTokenRingServer
	id                   int32
	enterCriticalSection bool
	clients              map[int32]ring.TokenRingClient
	ctx                  context.Context
}

func init() {
	// Generates a new seed on every program execution to get random number for when a process wants to enter the critical section
	rand.Seed(time.Now().UnixNano())
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:                   ownPort,
		enterCriticalSection: false,
		clients:              make(map[int32]ring.TokenRingClient),
		ctx:                  ctx,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	ring.RegisterTokenRingServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn

		log.Printf("Trying to dial: %v\n", port)

		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())

		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}

		defer conn.Close()

		c := ring.NewTokenRingClient(conn)

		p.clients[port] = c
	}

	// Calls function that decides when a process wants to enter the critical section
	go p.EnterCriticalSection()

	for {
		// Token ring starts with the process having port 5000
		if p.id == 5000 {
			time.Sleep(time.Second * 3)

			p.GetToken()
		}
	}
}

// Function that makes a process able to decide at any time if it wants to enter the critical section
func (p *peer) EnterCriticalSection() {

	n := 0

	// For loop that every 5 second randomly decides if the process wants to enter the critical section
	for n != 3 {
		time.Sleep(time.Second * 5)

		// Generate random number
		n = rand.Intn(6)

		// If the random number is 3, then the process wants to enter the critical section
		if n == 3 {
			log.Print("Process wants to enter critical section")

			p.enterCriticalSection = true
		}
	}
}

func (p *peer) SendToken(ctx context.Context, req *ring.Token) (*ring.Reply, error) {

	rep := &ring.Reply{Message: "Process received token!"}

	log.Print(rep.Message)

	// Wait 3 seconds before sending the token to the next in the ring
	time.Sleep(time.Second * 3)

	p.GetToken()

	return rep, nil
}

func (p *peer) GetToken() {

	// Check if process wants to enter the critical section
	if p.enterCriticalSection {

		// Process enters critical section
		p.CriticalSection()

		// Process does not want to enter critical section anymore
		p.enterCriticalSection = false

		// Wait 3 seconds before sending the token to the next in the ring
		time.Sleep(time.Second * 3)

		// Function that makes a process able to decide at any time if it wants to enter the critical section
		go p.EnterCriticalSection()
	}

	// Token to be sent to next process in token ring
	token := &ring.Token{ProcessId: p.id, TokenId: 1}

	// If process id is 5000 or 5001, send token to the next process by adding one to its own id
	if p.id == 5000 || p.id == 5001 {
		log.Printf("Process sends token to process with port: %v", p.id+1)
		p.clients[p.id+1].SendToken(p.ctx, token)
	} else {
		// If process id is 5002, send token to the first process in the token ring
		log.Printf("Process sends token to process with port: %v", 5000)
		p.clients[p.id-2].SendToken(p.ctx, token)
	}
}

// This is the function for the critical section
// As it's just a simulation it isn't a shared function between the processes
// However, it can still only be entered when a process decides it wants to and it has the token
func (p *peer) CriticalSection() {
	log.Print("Process entered critical section")

	// Process time in critical section
	time.Sleep(time.Second * 5)

	log.Print("Process exited critical section")
}
