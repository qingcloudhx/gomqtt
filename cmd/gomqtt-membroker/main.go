package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/qingcloudhx/gomqtt/broker"
	"github.com/qingcloudhx/gomqtt/packet"
	"github.com/qingcloudhx/gomqtt/transport"
)

var url = flag.String("url", "tcp://0.0.0.0:1883", "broker url")
var sqz = flag.Int("sqz", 100, "session queue size")

func main() {
	flag.Parse()

	go func() {
		panic(http.ListenAndServe("localhost:6060", nil))
	}()

	fmt.Printf("Starting broker on URL %s... ", *url)

	server, err := transport.Launch(*url)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done!")

	backend := broker.NewMemoryBackend()
	backend.SessionQueueSize = *sqz

	var published int32
	var forwarded int32
	var clients int32

	backend.Logger = func(event broker.LogEvent, client *broker.Client, pkt packet.Generic, msg *packet.Message, err error) {
		if event == broker.NewConnection {
			atomic.AddInt32(&clients, 1)
		} else if event == broker.MessagePublished {
			atomic.AddInt32(&published, 1)
		} else if event == broker.MessageForwarded {
			atomic.AddInt32(&forwarded, 1)
		} else if event == broker.LostConnection {
			atomic.AddInt32(&clients, -1)
		}
	}

	engine := broker.NewEngine(backend)
	engine.Accept(server)

	go func() {
		for {
			<-time.After(1 * time.Second)

			pub := atomic.LoadInt32(&published)
			fwd := atomic.LoadInt32(&forwarded)
			fmt.Printf("Publish Rate: %d msg/s, Forward Rate: %d msg/s, Clients: %d\n", pub, fwd, clients)

			atomic.StoreInt32(&published, 0)
			atomic.StoreInt32(&forwarded, 0)
		}
	}()

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT, syscall.SIGTERM)

	engine.OnError = func(err error) {
		fmt.Println(err.Error())
		finish <- nil
	}

	<-finish

	backend.Close(5 * time.Second)

	server.Close()

	engine.OnError = nil
	engine.Close()

	fmt.Println("Bye!")
}
