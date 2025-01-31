package spec

import (
	"time"

	"github.com/qingcloudhx/gomqtt/packet"
)

func safeReceive(ch chan struct{}) {
	select {
	case <-time.After(10 * time.Second):
		panic("nothing received")
	case <-ch:
	}
}

func lower(a, b packet.QOS) packet.QOS {
	if a < b {
		return a
	}

	return b
}
