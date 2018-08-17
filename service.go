package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Service struct {
	name     string
	key      string
	port     int32
	listener *net.TCPListener
}

func newService() *Service {
	p := &Service{}
	p.port = 0
	p.listener = nil

	return p
}

func (this *Service) loop() {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", this.port))
	if err != nil {
		log.Printf("net.ResolveTCPAddr: %v", err)
		return
	}
	this.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		log.Printf("net.ListenTCP: %v", err)
		return
	}

	log.Printf("listening on :%d for %s client ...\n", this.port, this.name)

	for {
		cn, err := this.listener.AcceptTCP()
		if err != nil {
			log.Printf("AcceptTCP: %v", err)
			time.Sleep(time.Second)
			continue
		}

		log.Printf("client(%s) has connected.\n", cn.RemoteAddr())

		var index int
		var session *Session
		g_mutexSession.Lock()
		for index, session = range g_session {
			if session.key == this.key {
				g_session = append(g_session[0:index], g_session[index+1:]...)
				break
			}
		}
		g_mutexSession.Unlock()

		if session == nil {
			log.Printf("agent has not connected.\n")
			cn.Close()
			continue
		}

		session.cnClient = cn
		r := session.start()
		if r {
			log.Printf("start session success\n")
		} else {
			log.Printf("start session fail\n")
		}
	}
}
