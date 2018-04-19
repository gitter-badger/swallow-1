package kvdb

type Session interface{}

type NodeSpec struct {
	ID      string            `json:"id"`
	Quota   map[string]uint16 `json:"quota"`
	Tags    []string          `json:"tags,omitempty"`
	Service string            `json:"service,"`
	Version uint8             `json:"version"`
}

type Leader struct {
	Add []*NodeSpec
	Del []*NodeSpec
	Chg []*NodeSpec
}

type Follower struct {
	LeaderID string
}

type Database interface {
	//manager
	StartElect() (<-chan Session, error)

	//slave
	Register(*NodeSpec) (<-chan Session, error)
}

func CreateDatabase(conf *Config) (db Database, err error) {

	switch conf.Type {
	case "zookeeper":
		{
			db, err = createZkDatabase(conf)
			if err != nil {
				return
			}
		}
	default:
		{
			panic("unexpect kv type")
		}
	}

	return
}
