package example

import (
	"errors"

	"github.com/luocheng812/swallow/kvdb"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNodeConfEmpty = errors.New("node config empty")
)

type Slave struct {
	db     kvdb.Database
	node   *kvdb.NodeSpec
	nodeID string
}

func NewSlave(conf *Config) (s *Slave, err error) {

	db, err := kvdb.CreateDatabase(conf.KvConf)

	if err != nil {
		return nil, err
	}

	return &Slave{
		db:     db,
		node:   conf.Node,
		nodeID: conf.ID,
	}, nil
}

func (s *Slave) logger() *log.Entry {
	return log.WithField("nodeID", s.nodeID)
}

func (s *Slave) Run() (err error) {

	if s.node == nil {
		s.logger().Warn(ErrNodeConfEmpty)
		return ErrNodeConfEmpty
	}

	session, err := s.db.Register(s.node)
	if err != nil {
		s.logger().Warn("register node:", err)
		return
	}

	for {
		_, ok := <-session
		if !ok {
			s.logger().Warn("node exit")
			break
		}
	}

	return
}
