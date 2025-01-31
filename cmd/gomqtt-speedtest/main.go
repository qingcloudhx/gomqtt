package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/qingcloudhx/gomqtt/client"
	"github.com/qingcloudhx/gomqtt/packet"

	"github.com/beorn7/perks/quantile"
)

var broker = flag.String("broker", "tcp://0.0.0.0:1883", "broker url")
var topic = flag.String("topic", "speed-test", "the used topic")
var qos = flag.Uint("qos", 0, "the qos level")
var wait = flag.Int("wait", 0, "time to wait in milliseconds")

var received = make(chan time.Time)

func main() {
	flag.Parse()

	cl := client.New()

	cl.Callback = func(msg *packet.Message, err error) error {
		if err != nil {
			panic(err)
		}

		received <- time.Now()
		return nil
	}

	cf, err := cl.Connect(client.NewConfig(*broker))
	if err != nil {
		panic(err)
	}

	err = cf.Wait(10 * time.Second)
	if err != nil {
		panic(err)
	}

	sf, err := cl.Subscribe(*topic, packet.QOS(*qos))
	if err != nil {
		panic(err)
	}

	err = sf.Wait(10 * time.Second)
	if err != nil {
		panic(err)
	}

	q := quantile.NewTargeted(map[float64]float64{
		0.50: 0.005,
		0.90: 0.001,
		0.99: 0.0001,
	})

	for {
		t1 := time.Now()

		pf, err := cl.Publish(*topic, []byte(*topic), packet.QOS(*qos), false)
		if err != nil {
			panic(err)
		}

		err = pf.Wait(10 * time.Second)
		if err != nil {
			panic(err)
		}

		t2 := <-received

		q.Insert(float64(t2.Sub(t1).Nanoseconds() / 1000 / 1000))

		q50 := time.Duration(q.Query(0.50)) * time.Millisecond
		q90 := time.Duration(q.Query(0.90)) * time.Millisecond
		q99 := time.Duration(q.Query(0.99)) * time.Millisecond

		fmt.Printf("[%d] 0.50: %s, 0.90: %s, 0.99: %s \n", q.Count(), q50, q90, q99)

		time.Sleep(time.Duration(*wait) * time.Millisecond)
	}
}
