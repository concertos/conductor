package conductor

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/client"
	"log"
	"sync"
	"time"
)

const ETCDENDPOINT = "http://127.0.0.1:2379"

type Conductor struct {
	Players map[string]*Player
	KeysAPI client.KeysAPI
}

type Player struct {
	Id       string
	Ips      []string
	Hostname string
	Cpu      int
	Online   bool
}

type PlayerInfo struct {
	Id       string
	Ips      []string
	Hostname string
	Cpu      int
}

func NodeToPlayerInfo(node *client.Node) *PlayerInfo {
	log.Println(node.Value)
	info := &PlayerInfo{}
	err := json.Unmarshal([]byte(node.Value), info)
	if err != nil {
		log.Print(err)
	}
	return info
}

func (c *Conductor) UpdatePlayer(info *PlayerInfo) {
	player := c.Players[info.Id]
	player.Online = true
}

func (c *Conductor) AddPlayer(info *PlayerInfo) {
	player := &Player{
		Online: true,
		Ips:    info.Ips,
		Id:     info.Id,
		Cpu:    info.Cpu,
	}
	c.Players[player.Id] = player
}

func (c *Conductor) Watch() {
	api := c.KeysAPI
	watcher := api.Watcher("/players/", &client.WatcherOptions{
		Recursive: true,
	})
	for {
		res, err := watcher.Next(context.Background())
		if err != nil {
			log.Println("Error watch workers:", err)
			break
		}
		if res.Action == "expire" {
			info := NodeToPlayerInfo(res.PrevNode)
			log.Println("Expire player ", info.Id)
			player, ok := c.Players[info.Id]
			if ok {
				player.Online = false
			}
		} else if res.Action == "set" {
			info := NodeToPlayerInfo(res.Node)
			if _, ok := c.Players[info.Id]; ok {
				log.Println("Update player ", info.Id)
				c.UpdatePlayer(info)
			} else {
				log.Println("Add player ", info.Id)
				c.AddPlayer(info)
			}
		} else if res.Action == "delete" {
			info := NodeToPlayerInfo(res.Node)
			log.Println("Delete player ", info.Id)
			delete(c.Players, info.Id)
		}
	}
}

func NewConductor() *Conductor {
	cfg := client.Config{
		Endpoints:               []string{ETCDENDPOINT},
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal("Error: cannot connec to etcd:", err)
	}

	conductor := &Conductor{
		Players: make(map[string]*Player),
		KeysAPI: client.NewKeysAPI(etcdClient),
	}
	//go conductor.Watch()
	return conductor
}

var conductor *Conductor
var once sync.Once

func GetConductor() *Conductor {
	once.Do(func() {
		conductor = NewConductor()
	})
	return conductor
}
