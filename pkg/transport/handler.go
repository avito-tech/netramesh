package transport

import (
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/log"
	"github.com/Lookyan/netramesh/pkg/protocol"
)

const SO_ORIGINAL_DST = 80

type TCPCopyBucket struct {
	R             io.ReadWriteCloser
	W             io.ReadWriteCloser
	Initiator     bool
	NetRequest    protocol.NetRequest
	NetHandler    protocol.NetHandler
	IsInBoundConn bool
	Done          chan struct{}
}

var tcpCopyBucketPool = sync.Pool{
	New: func() interface{} {
		return &TCPCopyBucket{}
	},
}

var addrPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 20)
	},
}

func TcpCopy(
	logger *log.Logger,
	r *net.TCPConn,
	w *net.TCPConn,
	initiator bool,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	isInBoundConn bool,
	f *os.File,
) {
	if initiator {
		netHandler.HandleRequest(r, w, netRequest, isInBoundConn)
	} else {
		netHandler.HandleResponse(r, w, netRequest, isInBoundConn)
	}
	f.Close()
	closeConn(logger, r)
	closeConn(logger, w)
	//done <- struct{}{}
}

func HandleConnection(
	logger *log.Logger,
	conn *net.TCPConn,
	ec *estabcache.EstablishedCache,
	tracingContextMapping *cache.Cache,
	//tcpCopyPool *ants.PoolWithFunc
) {
	if conn == nil {
		return
	}

	f, err := conn.File()
	if err != nil {
		closeConn(logger, conn)
		logger.Debug("Closed src conn")
		return
	}

	err = syscall.SetNonblock(int(f.Fd()), true)
	if err != nil {
		logger.Debug("Can't turn fd into non-blocking mode")
	}

	addr, err := syscall.GetsockoptIPv6Mreq(int(f.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		f.Close()
		closeConn(logger, conn)
		return
	}

	ipBuilder := addrPool.Get().([]byte)
	ipBuilder = append(ipBuilder, strconv.Itoa(int(addr.Multiaddr[4]))...)
	ipBuilder = append(ipBuilder, '.')
	ipBuilder = append(ipBuilder, strconv.Itoa(int(addr.Multiaddr[5]))...)
	ipBuilder = append(ipBuilder, '.')
	ipBuilder = append(ipBuilder, strconv.Itoa(int(addr.Multiaddr[6]))...)
	ipBuilder = append(ipBuilder, '.')
	ipBuilder = append(ipBuilder, strconv.Itoa(int(addr.Multiaddr[7]))...)
	ipv4 := string(ipBuilder)
	ipBuilder = ipBuilder[:0]
	addrPool.Put(ipBuilder)
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	isInBoundConn := ipv4 == strings.Split(conn.LocalAddr().String(), ":")[0]

	dstAddrBuilder := addrPool.Get().([]byte)
	dstAddrBuilder = append(dstAddrBuilder, ipv4...)
	dstAddrBuilder = append(dstAddrBuilder, ':')
	dstAddrBuilder = append(dstAddrBuilder, strconv.Itoa(int(port))...)
	dstAddr := string(dstAddrBuilder)
	dstAddrBuilder = dstAddrBuilder[:0]
	addrPool.Put(dstAddrBuilder)

	tcpDstAddr, err := net.ResolveTCPAddr("tcp", dstAddr)
	if err != nil {
		logger.Warningf("Error while resolving tcp addr %s", dstAddr)
	}
	targetConn, err := net.DialTCP("tcp", nil, tcpDstAddr)
	if err != nil {
		logger.Warning(err.Error())
		f.Close()
		closeConn(logger, conn)
		return
	}
	//defer func() {
	//	logger.Debug("Closing target conn")
	//	// same logic as for source tcp connection
	//	targetConn.CloseRead()
	//	targetConn.CloseWrite()
	//	targetConn.Close()
	//	logger.Debug("Closed target conn")
	//}()

	// determine protocol and choose logic
	p := protocol.Determine(dstAddr)
	netRequest := protocol.GetNetRequest(p, isInBoundConn, logger, tracingContextMapping)
	netHandler := protocol.GetNetworkHandler(p, logger, tracingContextMapping)

	//ec.Add(dstAddr)
	go TcpCopy(logger, conn, targetConn, true, netRequest, netHandler, isInBoundConn, f)
	go TcpCopy(logger, targetConn, conn, false, netRequest, netHandler, isInBoundConn, f)
	//<-done

	//ec.Remove(dstAddr)
}

func closeConn(logger *log.Logger, conn *net.TCPConn) {
	logger.Debug("Closing src conn")
	// Important to close read operations
	// to avoid waiting for never ending read operation when client doesn't close connection
	conn.CloseRead()
	conn.CloseWrite()
	conn.Close()
	logger.Debug("Closed src conn")
}
