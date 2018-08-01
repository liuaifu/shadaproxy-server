package main

import (
	"github.com/liuaifu/buffer"
	"log"
	"net"
	"sync"
	"time"
)

type Session struct {
	key        string
	cnClient   *net.TCPConn
	cnAgent    *net.TCPConn
	agentAddr  string
	clientAddr string
	onceStop   sync.Once
}

func newSession() *Session {
	p := &Session{}
	p.onceStop = sync.Once{}

	return p
}

func (this *Session) start() bool {
	if this.cnClient == nil {
		return false
	}
	if this.cnAgent == nil {
		return false
	}

	this.clientAddr = this.cnClient.RemoteAddr().String()

	head := &Head{}
	head.pkt_type = PKT_CONNECT
	log.Println("request agent connect to service")
	this.sendToAgent(head, nil)

	return true
}

func (this *Session) stop() {
	if this.cnClient != nil {
		this.cnClient.Close()
		this.cnClient = nil
		log.Printf("client has closed.")
	}

	if this.cnAgent != nil {
		this.cnAgent.Close()
		this.cnAgent = nil
		log.Printf("agent has closed.")
		//如果这个session还在g_session中，需要移除
		g_mutexSession.Lock()
		for index, v := range g_session {
			if this == v {
				g_session = append(g_session[:index], g_session[index+1:]...)
				break
			}
		}
		g_mutexSession.Unlock()
	}
}

func (this *Session) loop() {
	go this.agentLoop()
}

func (this *Session) clientLoop() {
	buf := make([]byte, 2048)
	for {
		if this.cnClient == nil {
			break
		}
		n, err := this.cnClient.Read(buf)
		if err != nil {
			log.Printf("client(%s) has closed!\n", this.clientAddr)
			break
		}

		head := &Head{}
		head.pkt_type = PKT_FORWARD_CLIENT_DATA
		body := buf[:n]
		if !this.sendToAgent(head, &body) {
			break
		}
	}

	this.onceStop.Do(this.stop)
}

func (this *Session) onHeartbeat() {
	head := &Head{}
	head.pkt_type = PKT_HEARTBEAT

	this.sendToAgent(head, nil)
}

func (this *Session) onSPMsg(head *Head, data []byte) bool {
	var msgType = head.pkt_type
	var msgLength = head.body_length
	var result = head.result

	switch msgType {
	case PKT_HEARTBEAT: //idle
		this.onHeartbeat()
		break
	case PKT_FORWARD_SERVER_DATA: //Server消息
		this.sendToClient(&data)
		break
	case PKT_REPORT_KEY:
		buf := &buffer.Buffer{data}
		key := buf.ReadStr()
		key_ok := false
		for _, service := range g_config.Services {
			if service.Key == key {
				key_ok = true
				this.key = key
				break
			}
		}
		head := &Head{}
		head.pkt_type = PKT_REPORT_KEY
		if !key_ok {
			log.Printf("key %s invalid\n", key)
			head.result = int32(0)
		} else {
			head.result = int32(1)
		}
		this.sendToAgent(head, nil)
		if !key_ok {
			this.onceStop.Do(this.stop)
		}
	case PKT_CONNECT:
		if result == 1 {
			go this.clientLoop()
			log.Printf("request connect server success\n")
		} else {
			log.Printf("request connect server fail\n")
			this.onceStop.Do(this.stop)
		}
		break
	default:
		log.Printf("unknown message Type=0x%X, Length=%d, Result=%d\n", msgType, msgLength, result)
		return false
	}

	return true
}

func (this *Session) agentLoop() {
	buf := []byte{}
	tmpBuf := make([]byte, 2048)
	for {
		if this.cnAgent == nil {
			break
		}
		this.cnAgent.SetReadDeadline(time.Now().Add(time.Minute))
		n, err := this.cnAgent.Read(tmpBuf)
		if err != nil {
			log.Printf("agentloop: %s!\n", err.Error())
			break
		}
		buf = append(buf, tmpBuf[:n]...)
		if len(buf) < 12 {
			continue
		}
		for len(buf) >= 12 {
			//检查是否有一条完整的消息
			var msgType, msgLength uint32
			msgType = uint32(buf[0])
			msgType |= uint32(buf[1]) << 8
			msgType |= uint32(buf[2]) << 16
			msgType |= uint32(buf[3]) << 24

			msgLength = uint32(buf[4])
			msgLength |= uint32(buf[5]) << 8
			msgLength |= uint32(buf[6]) << 16
			msgLength |= uint32(buf[7]) << 24

			if uint32(len(buf)) < (12 + msgLength) {
				break
			}

			var ret bool
			var result = int32(0)
			result = int32(buf[8])
			result |= int32(buf[9]) << 8
			result |= int32(buf[10]) << 16
			result |= int32(buf[11]) << 24
			head := &Head{msgType, msgLength, result}
			ret = this.onSPMsg(head, buf[12:12+msgLength])
			buf = buf[12+msgLength:]
			if !ret {
				log.Printf("warnning: invalid message! result=%d, Type=0X%X, Length=%d\n", result, msgType, msgLength)
			}
		}
	}

	this.onceStop.Do(this.stop)
}

/**
* 发送消息给服务代理
 */
func (this *Session) sendToAgent(head *Head, body *[]byte) bool {
	if this.cnAgent == nil || head == nil {
		this.onceStop.Do(this.stop)
		return false
	}

	buf := buffer.New()
	buf.WriteUint32(head.pkt_type) //类型
	if body == nil {
		buf.WriteUint32(0) //长度
	} else {
		buf.WriteUint32(uint32(len(*body))) //长度
	}
	buf.WriteInt32(head.result) //result
	if body != nil {
		buf.Append(*body)
	}

	data := buf.Buffer()
	leftCount := len(data)

	for leftCount > 0 {
		n, err := this.cnAgent.Write(data)
		if err != nil {
			log.Printf("send to %s: %s!\n", this.agentAddr, err)
			this.onceStop.Do(this.stop)
			return false
		}
		leftCount -= n
		data = data[n:]
	}

	return true
}

/**
* 转发来自服务的数据给用户端
 */
func (this *Session) sendToClient(data *[]byte) bool {
	leftCount := len(*data)
	if leftCount == 0 || this.cnClient == nil {
		this.onceStop.Do(this.stop)
		return false
	}

	for leftCount > 0 {
		n, err := this.cnClient.Write(*data)
		if err != nil {
			log.Printf("send to %s: %s!\n", this.clientAddr, err)
			this.onceStop.Do(this.stop)
			return false
		}
		leftCount -= n
		tmp_data := (*data)[n:]
		data = &tmp_data
	}

	return true
}
