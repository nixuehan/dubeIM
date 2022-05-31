package conf

import (
	"dube/pkg/otime"
	"flag"
	"github.com/BurntSushi/toml"
	"os"
)

type Options struct {
	WebSocket *WebSocket
	RPCClient *RPCClient
	RPCServer *RPCServer
	Env       *Env
	Bucket    *Bucket
}

type WebSocket struct {
	Bind            []string
	KeepAlive       bool
	ReadBufferSize  int
	WriteBufferSize int
}

type RPCClient struct {
	Dial    otime.Duration
	Timeout otime.Duration
}

type RPCServer struct {
	Dial    string
	Timeout string
}

type Bucket struct {
	Size    int32
	Channel int32
	Room    int32
}

type Env struct {
	Region string
	Zone   string
	Host   string
}

var (
	confPath   string
	region     string
	zone       string
	host       string
	defHost, _ = os.Hostname()
)

func init() {

	flag.StringVar(&confPath, "conf", "cmd/scratcher/scratcher.toml", "default config path.")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "available region. or use REGION env variable.etc: hn")
	flag.StringVar(&zone, "zone", os.Getenv("ZONE"), "available zone. or use ZONE env variable.etc: hn001")
	flag.StringVar(&host, "host", defHost, "server id.must be unique. default machine name")
}

func Default() (*Options, error) {
	options := &Options{
		Env: &Env{
			Region: region,
			Zone:   zone,
			Host:   host,
		},
	}
	_, err := toml.DecodeFile(confPath, &options)
	if err != nil {
		return nil, err
	}
	return options, nil
}
