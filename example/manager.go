package example

import (
	"github.com/luocheng812/swallow/kvdb"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	db        kvdb.Database
	managerID string
}

func NewManager(conf *Config) (mgr *Manager, err error) {

	db, err := kvdb.CreateDatabase(conf.KvConf)
	if nil != err {
		return nil, err
	}

	return &Manager{
		db:        db,
		managerID: conf.ID,
	}, nil
}

func (m *Manager) Run() (err error) {

	session, err := m.db.StartElect()

loopMain:
	for {
		s, ok := <-session
		if !ok {
			break loopMain
		}

		switch s.(type) {
		case *kvdb.Leader:
			{
				leader := s.(*kvdb.Leader)
				for _, node := range leader.Add {
					m.logger().Infof("add:%v", node)
				}
				for _, node := range leader.Del {
					m.logger().Infof("del:%v", node)
				}
				for _, node := range leader.Chg {
					m.logger().Infof("chg:%v", node)
				}
			}
		case *kvdb.Follower:
			{
				follower := s.(*kvdb.Follower)
				m.logger().Debug("be follower:", follower.LeaderID)
			}
		default:
			{
				m.logger().Panic("receive unexpect session", s)
			}
		}
	}
	return
}

func (m *Manager) Stop() {
}

func (m *Manager) logger() *log.Entry {
	return log.WithField("managerID", m.managerID)
}
