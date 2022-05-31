package scratcher

import (
	"context"
	"dube/internal/protocol"
	pb "dube/internal/protocol/cat"
	"dube/internal/scratcher/conf"
	"dube/pkg/websocket"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
	"math/rand"
	"sync"
	"time"
)

const (
	grpcInitialWindowSize     = 1 << 24
	grpcInitialConnWindowSize = 1 << 24
	grpcMaxCallMsgSize        = 1 << 24
	grpcMaxSendMsgSize        = 1 << 24
	grpcBackoffMaxDelay       = 3 * time.Second
	grpcKeepAliveTime         = 10 * time.Second
	grpcKeepAliveTimeout      = 3 * time.Second
)

const (
	MaxServerHeartbeat = 30 * time.Minute
	MinServerHeartbeat = 10 * time.Minute
)

type Channel struct {
	mid    int64
	key    string
	roomID string
	conn   *websocket.Conn
	q      chan string
	next   *Channel
}

func NewChannel() *Channel {
	return &Channel{}
}

type Room struct {
	Next *Channel
}

func NewRoom() *Room {
	return &Room{}
}

type Bucket struct {
	sync.RWMutex
	channelMap map[string]*Channel
	roomsMap   map[string]*Room
}

func NewBucket(c *conf.Bucket) *Bucket {
	return &Bucket{
		channelMap: make(map[string]*Channel, c.Channel),
		roomsMap:   make(map[string]*Room, c.Room),
	}
}

// put channel
func (b *Bucket) put(ch *Channel) error {
	b.Lock()
	b.channelMap[ch.key] = ch

	if ch.roomID != "" {

		if _, ok := b.roomsMap[ch.roomID]; !ok {
			b.roomsMap[ch.roomID] = NewRoom()
			b.roomsMap[ch.roomID].Next = ch
		} else {
			ch.next = b.roomsMap[ch.roomID].Next
			b.roomsMap[ch.roomID].Next = ch
		}
	}
	b.Unlock()
	return nil
}

func (b *Bucket) get(key string) (*Channel, error) {
	b.RLock()
	v, ok := b.channelMap[key]
	b.RUnlock()
	if ok {
		return v, nil
	}
	return nil, fmt.Errorf("get key(%s) in bucket error", key)
}

func (b *Bucket) del(ch *Channel) {
	b.Lock()
	defer b.Unlock()

	delete(b.channelMap, ch.key)

	if _, ok := b.roomsMap[ch.roomID]; ok {
		delete(b.roomsMap, ch.roomID)
	}
}

func (b *Bucket) room(rid string) *Room {
	b.RLock()
	defer b.RUnlock()
	return b.roomsMap[rid]
}

type Scratcher struct {
	Conf          *conf.Options
	RpcClient     pb.CatClient
	ServerID      string
	Bucket        *Bucket
	LastHeartbeat time.Time
}

func NewRPCClient(c *conf.RPCClient) pb.CatClient {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Dial))
	defer cancel()

	cc, err := grpc.DialContext(ctx, "127.0.0.1:3119",
		[]grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithInitialWindowSize(grpcInitialWindowSize),
			grpc.WithInitialConnWindowSize(grpcInitialConnWindowSize),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcMaxCallMsgSize)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(grpcMaxSendMsgSize)),
			grpc.WithBackoffMaxDelay(grpcBackoffMaxDelay),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                grpcKeepAliveTime,
				Timeout:             grpcKeepAliveTimeout,
				PermitWithoutStream: true,
			}),
			grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, roundrobin.Name)),
		}...)
	if err != nil {
		panic(err)
	}

	return pb.NewCatClient(cc)
}

func New(c *conf.Options) *Scratcher {
	return &Scratcher{
		c,
		NewRPCClient(c.RPCClient),
		c.Env.Host,
		NewBucket(c.Bucket),
		time.Now(),
	}
}

func (s *Scratcher) Auth(ctx context.Context, p *protocol.Protocol, conn *websocket.Conn) (mid int64, key, roomID string, hb int64, err error) {

	if err = p.ReadWebsocket(conn); err != nil {
		return
	}

	resp, err := s.RpcClient.Identify(ctx, &pb.IdentifyReq{
		Server: s.ServerID,
		Token:  p.Body,
	})

	if err != nil {
		return
	}

	p.Op = protocol.OpAuthReply
	if err = p.WriteWebsocket(conn); err != nil {
		return
	}

	return resp.Mid, resp.Key, resp.RoomID, resp.Heartbeat, nil
}

func (s *Scratcher) Handle(p *protocol.Protocol) error {
	return p.Exec()
}

func (s *Scratcher) RandServerHeartbeat() time.Duration {
	return MinServerHeartbeat + time.Duration(rand.Int63n(int64(MaxServerHeartbeat-MinServerHeartbeat)))
}

func (s *Scratcher) Dispatch(ctx context.Context, channel *Channel) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-channel.q:
			fmt.Println(c)
		}
	}
}
