package example

import (
	"encoding/json"
	"io/ioutil"

	"github.com/luocheng812/swallow/kvdb"
)

type Config struct {
	ID     string         `json:"id"`
	KvConf *kvdb.Config   `json:"kvdb"`
	Node   *kvdb.NodeSpec `json:"node,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		ID:     "ID",
		KvConf: kvdb.DefaultConfig(),
		Node:   &kvdb.NodeSpec{},
	}
}

func NewConfig(confPath string) (conf *Config, err error) {

	conf = &Config{}
	err = conf.Load(confPath)
	return
}

func (c *Config) Load(conf string) (err error) {

	var buf []byte
	if buf, err = ioutil.ReadFile(conf); nil != err {
		return
	}

	err = json.Unmarshal(buf, c)
	return
}
