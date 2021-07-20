package libp2pquic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"

	ic "github.com/libp2p/go-libp2p-core/crypto"
	tpt "github.com/libp2p/go-libp2p-core/transport"
	"github.com/lucas-clemente/quic-go"

	ma "github.com/multiformats/go-multiaddr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// interface containing some methods defined on the net.UDPConn, but not the net.PacketConn
type udpConn interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	SetReadBuffer(bytes int) error
	SyscallConn() (syscall.RawConn, error)
}

var _ = Describe("Listener", func() {
	var t tpt.Transport

	BeforeEach(func() {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())
		key, err := ic.UnmarshalRsaPrivateKey(x509.MarshalPKCS1PrivateKey(rsaKey))
		Expect(err).ToNot(HaveOccurred())
		t, err = NewTransport(key, nil, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(t.(io.Closer).Close()).To(Succeed())
	})

	It("uses a conn that can interface assert to a UDPConn for listening", func() {
		origQuicListen := quicListen
		defer func() { quicListen = origQuicListen }()

		var conn net.PacketConn
		quicListen = func(c net.PacketConn, _ *tls.Config, _ *quic.Config) (quic.Listener, error) {
			conn = c
			return nil, errors.New("listen error")
		}
		localAddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/quic")
		Expect(err).ToNot(HaveOccurred())
		_, err = t.Listen(localAddr)
		Expect(err).To(MatchError("listen error"))
		Expect(conn).ToNot(BeNil())
		defer conn.Close()
		_, ok := conn.(udpConn)
		Expect(ok).To(BeTrue())
	})

	Context("listening on the right address", func() {
		It("returns the address it is listening on", func() {
			localAddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/quic")
			Expect(err).ToNot(HaveOccurred())
			ln, err := t.Listen(localAddr)
			Expect(err).ToNot(HaveOccurred())
			defer ln.Close()
			netAddr := ln.Addr()
			Expect(netAddr).To(BeAssignableToTypeOf(&net.UDPAddr{}))
			port := netAddr.(*net.UDPAddr).Port
			Expect(port).ToNot(BeZero())
			Expect(ln.Multiaddr().String()).To(Equal(fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", port)))
		})

		It("returns the address it is listening on, for listening on IPv4", func() {
			localAddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/udp/0/quic")
			Expect(err).ToNot(HaveOccurred())
			ln, err := t.Listen(localAddr)
			Expect(err).ToNot(HaveOccurred())
			defer ln.Close()
			netAddr := ln.Addr()
			Expect(netAddr).To(BeAssignableToTypeOf(&net.UDPAddr{}))
			port := netAddr.(*net.UDPAddr).Port
			Expect(port).ToNot(BeZero())
			Expect(ln.Multiaddr().String()).To(Equal(fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port)))
		})

		It("returns the address it is listening on, for listening on IPv6", func() {
			localAddr, err := ma.NewMultiaddr("/ip6/::/udp/0/quic")
			Expect(err).ToNot(HaveOccurred())
			ln, err := t.Listen(localAddr)
			Expect(err).ToNot(HaveOccurred())
			defer ln.Close()
			netAddr := ln.Addr()
			Expect(netAddr).To(BeAssignableToTypeOf(&net.UDPAddr{}))
			port := netAddr.(*net.UDPAddr).Port
			Expect(port).ToNot(BeZero())
			Expect(ln.Multiaddr().String()).To(Equal(fmt.Sprintf("/ip6/::/udp/%d/quic", port)))
		})
	})

	Context("accepting connections", func() {
		var localAddr ma.Multiaddr

		BeforeEach(func() {
			var err error
			localAddr, err = ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/quic")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns Accept when it is closed", func() {
			addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/quic")
			Expect(err).ToNot(HaveOccurred())
			ln, err := t.Listen(addr)
			Expect(err).ToNot(HaveOccurred())
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				ln.Accept()
				close(done)
			}()
			Consistently(done).ShouldNot(BeClosed())
			Expect(ln.Close()).To(Succeed())
			Eventually(done).Should(BeClosed())
		})

		It("doesn't accept Accept calls after it is closed", func() {
			ln, err := t.Listen(localAddr)
			Expect(err).ToNot(HaveOccurred())
			Expect(ln.Close()).To(Succeed())
			_, err = ln.Accept()
			Expect(err).To(HaveOccurred())
		})
	})
})
