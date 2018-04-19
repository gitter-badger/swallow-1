package kvdb

import (
	"bytes"
	"encoding/json"
	"sync/atomic"
	"time"

	leaderelection "github.com/Comcast/go-leaderelection"
	"github.com/samuel/go-zookeeper/zk"

	log "github.com/sirupsen/logrus"
)

const (
	zkHeartBeat = 10 * time.Second
	zkKeepalive = 60 * time.Second
)

type zkSession struct {
	conn      *zk.Conn
	event     <-chan zk.Event
	endpoints []string
}

func createZkSession(endpoints []string) (*zkSession, error) {

	zkConn, zkEvent, err := zk.Connect(endpoints, zkHeartBeat)
	if err != nil {
		log.Errorf("zk.Connect(%s):%v", endpoints, err)
		return nil, err
	}

	return &zkSession{
		conn:      zkConn,
		event:     zkEvent,
		endpoints: endpoints,
	}, nil

}

func (z *zkSession) logger() *log.Entry {
	return log.WithField("zk:", z.conn.Server())
}

func (z *zkSession) status() <-chan zk.Event {
	return z.event
}

func (z *zkSession) createNodeIfNotExist(path string, value string) error {

	_, err := z.conn.Create(path, []byte(value), 0, zk.WorldACL(zk.PermAll))

	if err != nil && err != zk.ErrNodeExists {
		z.logger().Errorf("create node (%s):%v", path, err)
		return err
	}

	return nil
}

func (z *zkSession) updateChildrenStatus(path string, watch bool) ([]string, <-chan zk.Event, error) {

	var data []string
	var stat *zk.Stat
	var nodeChange <-chan zk.Event
	var err error

	if watch {
		data, stat, nodeChange, err = z.conn.ChildrenW(path)
	} else {
		data, stat, err = z.conn.Children(path)
	}

	if nil != err {
		z.logger().Warnf("update child status data(%s) stat(%v) nodeChange(%v) err(%v)", data, stat, err)
		return nil, nil, err
	}

	return data, nodeChange, err
}

func (z *zkSession) updateNodeStatus(path string, watch bool) ([]byte, <-chan zk.Event, error) {

	var data []byte
	var stat *zk.Stat
	var change <-chan zk.Event
	var err error

	if watch {
		data, stat, change, err = z.conn.GetW(path)
	} else {
		data, stat, err = z.conn.Get(path)
	}

	if nil != err {
		z.logger().Warnf("update node status data(%s) stat(%v) nodeChange(%v) err(%v)", data, stat, err)
		return nil, nil, err
	}

	return data, change, err
}

type zkDatabase struct {
	session *zkSession
	conf    *Config
	elected int32
	nodes   map[string][]byte
}

func createZkDatabase(conf *Config) (*zkDatabase, error) {

	session, err := createZkSession(conf.EndPoints)

	if nil != err {
		log.Errorf("new zookeeper session endpoints(%v):(%v)", conf.EndPoints, err)
		return nil, err
	}

	if err := session.createNodeIfNotExist(conf.getRootNode(), ""); nil != err {
		log.Warnf("create node <%s> :%v", conf.getRootNode(), err)
		return nil, err
	}

	if err := session.createNodeIfNotExist(conf.getProjectNode(), ""); nil != err {
		log.Warnf("create node <%s> :%v", conf.getProjectNode(), err)
		return nil, err
	}

	if err := session.createNodeIfNotExist(conf.getTaskSpecPath(), ""); nil != err {
		log.Warnf("create node <%s> :%v", conf.getTaskSpecPath(), err)
		return nil, err
	}

	if err := session.createNodeIfNotExist(conf.getNodeSpecPath(), ""); nil != err {
		log.Warnf("create node <%s> :%v", conf.getNodeSpecPath(), err)
		return nil, err
	}

	if err := session.createNodeIfNotExist(conf.getElectNode(), ""); nil != err {
		log.Warnf("create node <%s> :%v", conf.getElectNode(), err)
		return nil, err
	}

	return &zkDatabase{
		session: session,
		conf:    conf,
		elected: 0,
		nodes:   make(map[string][]byte),
	}, nil
}

func (z *zkDatabase) logger() *log.Entry {
	return log.WithField("zk", z.session.conn.Server())
}

func (z *zkDatabase) StartElect() (<-chan Session, error) {

	candidate, err := leaderelection.NewElection(z.session.conn, z.conf.getElectNode(), z.conf.getLeaderID())
	if nil != err {
		z.logger().Error("leaderelection :", err)
		return nil, err
	}

	session := make(chan Session, 10)

	go candidate.ElectLeader()

	go z.waitBecomeLeader(candidate, session)

	return session, nil
}

func (z *zkDatabase) Register(node *NodeSpec) (<-chan Session, error) {

	data, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	nodePath := z.conf.getNodeSpecPath() + "/" + node.ID
	_, err = z.session.conn.Create(nodePath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))

	session := make(chan Session, 10)

	go func() {

		for {
			ev := <-z.session.status()
			z.logger().Debug("receive event:", ev)
			if ev.Err != nil {
				break
			}
		}

		close(session)

	}()

	return session, err
}

func (z *zkDatabase) handleLeaderChange() (<-chan zk.Event, *Follower, error) {

	childrens, change, err := z.session.updateChildrenStatus(z.conf.getElectNode(), true)

	if nil != err || len(childrens) == 0 {
		z.logger().Warn("session update children status:", err, " candidateID:", childrens)
		return nil, nil, err
	}

	follower := &Follower{}

	// the last elect node was be leader
	data, _, err := z.session.updateNodeStatus(z.conf.getElectNode()+"/"+childrens[0], false)

	if nil != err {
		z.logger().Warnf("session update children data(%s):%v", childrens[0], err)
		return nil, nil, err
	}

	follower.LeaderID = string(data)

	return change, follower, err

}

func (z *zkDatabase) handleNodeChange() (<-chan zk.Event, *Leader, error) {

	childrens, change, err := z.session.updateChildrenStatus(z.conf.getNodeSpecPath(), true)

	if nil != err {
		z.logger().Warn("session update children status:", err)
		return nil, nil, err
	}

	nodes := make(map[string][]byte)

	for _, children := range childrens {

		data, _, err := z.session.updateNodeStatus(z.conf.getNodeSpecPath()+"/"+children, false)

		if nil != err {
			z.logger().Warnf("session update children data(%s):", children, err)
			continue
		}

		nodes[children] = data
	}

	leader := &Leader{}
	leader.Add, leader.Del, leader.Chg = z.diffNodeStatus(z.nodes, nodes)

	z.nodes = nodes

	return change, leader, err

}

func (z *zkDatabase) diffNodeStatus(old map[string][]byte, new map[string][]byte) (add []*NodeSpec, del []*NodeSpec, chg []*NodeSpec) {

	for k, v := range old {
		if _, ok := new[k]; !ok {
			nodeSpec := &NodeSpec{}
			if err := json.Unmarshal(v, nodeSpec); err != nil {
				z.logger().Warnf("json nodespec unmarshal(%s):%v", string(v), err)
			} else {
				del = append(del, nodeSpec)
			}
		}
	}

	for newK, newV := range new {

		if oldV, ok := old[newK]; !ok {

			nodeSpec := &NodeSpec{}
			if err := json.Unmarshal(newV, nodeSpec); err != nil {
				z.logger().Warnf("json nodespec unmarshal(%s):%v", string(newV), err)
			} else {
				add = append(add, nodeSpec)
			}

		} else if !bytes.Equal(newV, oldV) {

			nodeSpec := &NodeSpec{}
			if err := json.Unmarshal(newV, nodeSpec); err != nil {
				z.logger().Warnf("json nodespec unmarshal(%s):%v", string(newV), err)
			} else {
				chg = append(chg, nodeSpec)
			}
		}
	}

	return
}

func (z *zkDatabase) amILeader() bool {
	return 1 == atomic.LoadInt32(&z.elected)
}

func (z *zkDatabase) waitBecomeLeader(candidate *leaderelection.Election, session chan<- Session) {

	var tick <-chan time.Time
	var nodeChange <-chan zk.Event
	var leaderChange <-chan zk.Event

loopWait:
	for {
		select {
		case ev := <-z.session.status():
			{
				z.logger().Debug("receive event:", ev)
				if ev.Err != nil {
					break loopWait
				}
			}
		case status, ok := <-candidate.Status():
			{
				if !ok || nil != status.Err { //channel close ,
					z.logger().Warn("candidate status:", status, ok)
					break loopWait
				}

				if !z.amILeader() && (status.Role == leaderelection.Leader) {

					z.logger().Debug("become leader")

					if change, leader, err := z.handleNodeChange(); err == nil {
						nodeChange = change
						session <- leader
						atomic.StoreInt32(&z.elected, 1)
						tick = time.NewTicker(zkKeepalive).C
					} else {
						z.logger().Warn("watch data change:", err)
						break loopWait
					}

				} else if status.Role == leaderelection.Follower {

					if z.amILeader() {
						z.logger().Warn("leader change follower:", status)
						break loopWait
					}

					if change, follower, err := z.handleLeaderChange(); err == nil {
						leaderChange = change
						session <- follower
						z.logger().Info("now folloer leader:", follower)
					} else {
						z.logger().Warn("get leader id:", err)
						break loopWait
					}

				} else {
					z.logger().Warn("elect status unexpection:", status)
					break loopWait
				}
			}
		case <-nodeChange:
			{
				if change, leader, err := z.handleNodeChange(); err != nil {
					z.logger().Warn("handle node change:", err)
				} else {
					nodeChange = change
					session <- leader
				}
			}
		case <-leaderChange:
			{
				if !z.amILeader() {
					if change, follower, err := z.handleLeaderChange(); err != nil {
						z.logger().Warn("handle node change:", err)
					} else {
						leaderChange = change
						session <- follower
					}
				}
			}
		case <-tick:
			{
				// check zookeeper alive
				if _, _, err := z.session.updateChildrenStatus(z.conf.getElectNode(), false); err != nil {
					z.logger().Warn("zookeeper elect node checkalive:", err)
					break loopWait
				}
			}
		}
	}

	candidate.Resign()
	close(session)
}
