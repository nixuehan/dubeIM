package options

import (
	"dube/pkg/otime"
	"flag"
	"github.com/BurntSushi/toml"
	"os"
	"time"
)

type Options struct {
	Env        *Env
	RpcServer  *RpcServer
	Redis      *Redis
	Node       *Node
	HTTPServer *HTTPServer
}

type Node struct {
	Domain    string
	Heartbeat otime.Duration
	Weight    float64
}

type RpcServer struct {
	Network           string
	Addr              string
	Timeout           otime.Duration
	IdleTimeout       otime.Duration
	MaxLifeTime       otime.Duration
	ForceCloseWait    otime.Duration
	KeepAliveInterval otime.Duration
	KeepAliveTimeout  otime.Duration
}

type HTTPServer struct {
	Network      string
	Addr         string
	ReadTimeout  otime.Duration
	WriteTimeout otime.Duration
}

type Kafka struct {
	topic   string
	brokers []string
}

type Redis struct {
	Network      string
	Addr         string
	Auth         string //TODO from deploy env.
	Active       int
	Idle         int
	DialTimeout  otime.Duration
	ReadTimeout  otime.Duration
	WriteTimeout otime.Duration
	IdleTimeout  otime.Duration
	Expire       otime.Duration
}

type Env struct {
	Region string
	Zone   string
	Host   string
}

var (
	confPath string
	region   string
	zone     string
	host     string
)

func init() {
	var (
		hostname, _ = os.Hostname()
	)
	flag.Parse()
	flag.StringVar(&confPath, "conf", "cmd/cat/cat.toml", "default config path.")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "available region. default REGION env variable. value: sh etc.")
	flag.StringVar(&zone, "zone", os.Getenv("ZONE"), "available region. default ZONE env variable. value sh001 etc.")
	flag.StringVar(&host, "host", hostname, "machine hostname.")
}

func InitOptions() (*Options, error) {
	o := Default()
	if _, err := toml.DecodeFile(confPath, &o); err != nil {
		return nil, err
	}
	return o, nil
}

func Default() *Options {
	return &Options{
		Env: &Env{Region: region, Zone: zone, Host: host},
		RpcServer: &RpcServer{
			Network:           "tcp",
			Addr:              ":3319",
			Timeout:           otime.Duration(time.Second),
			IdleTimeout:       otime.Duration(time.Second * 60),
			MaxLifeTime:       otime.Duration(time.Hour * 2),
			ForceCloseWait:    otime.Duration(time.Second * 20),
			KeepAliveInterval: otime.Duration(time.Second * 60),
			KeepAliveTimeout:  otime.Duration(time.Second * 20),
		},
	}
}
