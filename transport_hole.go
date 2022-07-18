package libp2pquic

import (
	"crypto/rand"
	"net"
	"time"
)

func (t *transport) preHole(pconn *reuseConn, addr *net.UDPAddr) error {
	var (
		punchErr error
		payload  = make([]byte, 64)
	)
	for i := 0; i < 30; i++ {
		if _, err := rand.Read(payload); err != nil {
			punchErr = err
			break
		}
		if _, err := pconn.UDPConn.WriteToUDP(payload, addr); err != nil {
			punchErr = err
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	return punchErr
}
