package main

import (
	"encoding/xml"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type ServiceCfg struct {
	Name string `xml:"name"`
	Key  string `xml:"key"`
	Port int    `xml:"port"`
}

type Config struct {
	XMLName      xml.Name     `xml:"config"`
	PortForAgent int          `xml:"port_for_agent"`
	Services     []ServiceCfg `xml:"services>service"`
}

var g_config_file string
var g_daemon bool
var g_config *Config
var g_session []*Session //空闲的agent会话
var g_mutexSession *sync.RWMutex

func init() {
	execPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Fatal(err)
	}
	//Is Symlink
	fi, err := os.Lstat(execPath)
	if err != nil {
		log.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		execPath, err = os.Readlink(execPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	execDir := filepath.Dir(execPath)
	if execDir == "." {
		execDir, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Chdir(execDir)
	flag.StringVar(&g_config_file, "config", filepath.Join(execDir, "config.xml"), "config file")
	if runtime.GOOS != "windows" {
		flag.BoolVar(&g_daemon, "d", false, "run as daemon")
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds /* | log.Lshortfile*/)

	g_session = []*Session{}
	g_mutexSession = new(sync.RWMutex)
}

func main() {
	flag.Parse()

	if !g_daemon || os.Getppid() != 1 {
		println("shadaproxy-server v0.4")
		println("copyright(c) 2011-2018 laf163@gmail.com\n")
	}

	if g_daemon && os.Getppid() != 1 {
		//不是子进程
		filePath, _ := filepath.Abs(os.Args[0])
		cmd := exec.Command(filePath, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()

		println("run in background")

		return
	}

	logFile, err := os.OpenFile("./shadaproxy-server.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	defer logFile.Close()
	if err == nil {
		log.SetOutput(logFile)
	} else {
		log.Println(err)
	}
	log.Println("---------------------------------------------------------------")

	data, err := ioutil.ReadFile(g_config_file)
	if err != nil {
		panic(err)
	}
	g_config = &Config{}
	err = xml.Unmarshal(data, g_config)
	if err != nil {
		panic(err)
	}

	for _, serviceCfg := range g_config.Services {
		service := newService()
		service.name = serviceCfg.Name
		service.key = serviceCfg.Key
		service.port = int32(serviceCfg.Port)
		go service.loop()
	}

	agent := newAgent()
	agent.port = int32(g_config.PortForAgent)
	agent.loop()
}
