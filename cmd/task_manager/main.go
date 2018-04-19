package main

import (
	"flag"

	"github.com/luocheng812/swallow/example"
	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
)

var (
	conf = flag.String("conf", "./conf/manager.json", "manager config file")
)

func main() {

	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.AddHook(filename.NewHook())

	config, err := example.NewConfig(*conf)
	if err != nil {
		log.Error("new conf:", err)
		return
	}

	mgr, err := example.NewManager(config)
	if nil != err {
		log.Error("new manager:", err)
		return
	}

	err = mgr.Run()
	if nil != err {
		log.Error("manager run:", err)
		return
	}

	log.Debug("hello", mgr, err)

}
