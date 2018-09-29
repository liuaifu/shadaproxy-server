package main

import (
	"os"
	"fmt"
	"log"
	"net"
)

type agent struct {
	port     int32
	listener *net.TCPListener
}

func newAgent() *agent {
	p := &agent{}
	p.port = 0
	p.listener = nil

	return p
}

func (this *agent) loop() {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", this.port))
	if err != nil {
		log.Printf("net.ResolveTCPAddr: %v", err)
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	this.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		log.Printf("net.ListenTCP: %v", err)
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	log.Printf("listening on :%d for agent ...\n", this.port)

	for {
		c, err := this.listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %v", err)
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		log.Printf("agent(%s) has connected.\n", c.RemoteAddr())
		go this.createSession(c)
	}
}

func (this *agent) createSession(cn *net.TCPConn) {
	p := newSession()
	p.cnAgent = cn
	p.agentAddr = cn.RemoteAddr().String()

	g_mutexSession.Lock()
	g_session = append(g_session, p)
	g_mutexSession.Unlock()

	p.loop()
}
