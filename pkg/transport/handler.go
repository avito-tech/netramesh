package transport

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/protocol"
)

const SO_ORIGINAL_DST = 80

// tcpRequestCopy returns new address to go if we don't go to original DST
func tcpRequestCopy(
	r io.ReadWriteCloser,
	w io.ReadWriter,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	isInBoundConn bool) string {
	return netHandler.HandleRequest(r, w, netRequest, isInBoundConn)
}

// tcpRequestCopy returns new address to go if we don't go to original DST
func tcpResponseCopy(
	r io.ReadWriteCloser,
	w io.ReadWriteCloser,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	isInBoundConn bool,
	done chan struct{}) {
	netHandler.HandleResponse(r, w, netRequest, isInBoundConn)
	done <- struct {}{}
}

// HandleConnection handles netra input connections
func HandleConnection(conn *net.TCPConn, ec *estabcache.EstablishedCache, tracingContextMapping *cache.Cache) {
	if conn == nil {
		return
	}
	defer func() {
		log.Print("Closing src conn")
		// Important to close read operations
		// to avoid waiting for never ending read operation when client doesn't close connection
		conn.CloseRead()
		conn.CloseWrite()
		conn.Close()
		log.Print("Closed src conn")
	}()
	conn.SetNoDelay(true)

	f, err := conn.File()
	if err != nil {
		log.Print(err.Error())
		return
	}
	defer f.Close()
	err = syscall.SetNonblock(int(f.Fd()), true)
	if err != nil {
		log.Print("Can't turn fd into non-blocking mode")
	}

	addr, err := syscall.GetsockoptIPv6Mreq(int(f.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		log.Print(err.Error())
		return
	}
	ipv4 := strconv.Itoa(int(addr.Multiaddr[4])) + "." +
		strconv.Itoa(int(addr.Multiaddr[5])) + "." +
		strconv.Itoa(int(addr.Multiaddr[6])) + "." +
		strconv.Itoa(int(addr.Multiaddr[7]))
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	isInBoundConn := ipv4 == strings.Split(conn.LocalAddr().String(), ":")[0]

	dstAddr := fmt.Sprintf("%s:%d", ipv4, port)
	log.Printf("From: %s To: %s", conn.RemoteAddr(), conn.LocalAddr())
	log.Printf("Original destination :: %s", dstAddr)

	// determine protocol and choose logic
	p := protocol.Determine(dstAddr)
	log.Printf("Determined %s protocol", p)
	netRequest := protocol.GetNetRequest(p)
	netHandler := protocol.GetNetworkHandler(p, tracingContextMapping)

	// waiting while interaction ended
	done := make(chan struct{}, 1)

	requestReader := &bytes.Buffer{}
	// protocol can change final destination
	newDstAddr := tcpRequestCopy(conn, requestReader, netRequest, netHandler, isInBoundConn)
	if newDstAddr != "" {
		dstAddr = newDstAddr
	}

	tcpDstAddr, err := net.ResolveTCPAddr("tcp", dstAddr)
	if err != nil {
		log.Printf("Error while resolving tcp addr %s", dstAddr)
	}
	targetConn, err := net.DialTCP("tcp", nil, tcpDstAddr)
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer func() {
		log.Print("Closing target conn")
		// same logic as for source tcp connection
		targetConn.CloseRead()
		targetConn.CloseWrite()
		targetConn.Close()
		log.Print("Closed target conn")
	}()

	ec.Add(dstAddr)

	go func() {
		io.Copy(targetConn, requestReader)
		done <- struct{}{}
	}()
	go tcpResponseCopy(targetConn, conn, netRequest, netHandler, isInBoundConn, done)

	<-done

	log.Print("Finished")
	ec.Remove(dstAddr)
}
