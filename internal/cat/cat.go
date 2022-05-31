package cat

import (
	"context"
	"dube/internal/cat/dao"
	"dube/internal/cat/options"
	"encoding/json"
	log "github.com/golang/glog"
	"github.com/google/uuid"
)

type Cat struct {
	dao  *dao.Dao
	node *options.Node
}

func New(c *options.Options) *Cat {
	return &Cat{
		dao.New(c.Redis),
		c.Node,
	}
}

func (c *Cat) PutKeys(op int32, keys []string, data []byte) error {

	return nil
}

func (c *Cat) Heartbeat(ctx context.Context, mid int64, key, server string) error {
	if has, _ := c.dao.ExpireMapping(mid, key); !has {
		if err := c.dao.AddMapping(mid, key, server); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cat) Identify(ctx context.Context, server string, token []byte) (mid int64, key, roomID string, hb int64, err error) {
	var p struct {
		Mid      int64  `json:"Mid"`
		Key      string `json:"Key"`
		RoomID   string `json:"room_id"`
		Platform string `json:"Platform"`
	}

	if err = json.Unmarshal(token, &p); err != nil {
		log.Errorf("json.Unmarshal(%s) error - (%v)", err)
		return
	}

	mid = p.Mid
	key = p.Key
	roomID = p.RoomID
	hb = int64(c.node.Heartbeat)

	if key == "" {
		key = uuid.New().String()
	}

	if err = c.dao.AddMapping(mid, key, server); err != nil {
		return
	}

	return
}
