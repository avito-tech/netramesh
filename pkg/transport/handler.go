package transport

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/internal/config"
	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/log"
	"github.com/Lookyan/netramesh/pkg/protocol"
)

const SO_ORIGINAL_DST = 80

var addrPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 20)
	},
}

func TcpCopyRequest(
	logger *log.Logger,
	r *net.TCPConn,
	w *net.TCPConn,
	connCh chan *net.TCPConn,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	isInBoundConn bool,
	f *os.File,
	addrCh chan string,
	originalDst string,
) {
	w = netHandler.HandleRequest(r, w, connCh, addrCh, netRequest, isInBoundConn, originalDst)
	f.Close()
	closeConn(logger, r)
	if w != nil {
		closeConn(logger, w)
	}
}

func TcpCopyResponse(
	logger *log.Logger,
	r *net.TCPConn,
	w *net.TCPConn,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	isInBoundConn bool,
	f *os.File,
) {
	netHandler.HandleResponse(r, w, netRequest, isInBoundConn)
	f.Close()
	closeConn(logger, r)
	closeConn(logger, w)
}

func HandleConnection(
	logger *log.Logger,
	conn *net.TCPConn,
	ec *estabcache.EstablishedCache,
	tracingContextMapping *cache.Cache,
	routingInfoContextMapping *cache.Cache,
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
	originalDstAddr := string(dstAddrBuilder)
	dstAddrBuilder = dstAddrBuilder[:0]
	addrPool.Put(dstAddrBuilder)

	// determine protocol and choose logic
	p := protocol.Determine(originalDstAddr)
	netRequest := protocol.GetNetRequest(p, isInBoundConn, logger, tracingContextMapping)
	netHandler := protocol.GetNetworkHandler(p, logger, tracingContextMapping)

	//ec.Add(dstAddr)
	if config.GetHTTPConfig().RoutingEnabled {
		addrCh := make(chan string)
		connCh := make(chan *net.TCPConn)
		go TcpCopyRequest(
			logger,
			conn,
			nil,
			connCh,
			netRequest,
			netHandler,
			isInBoundConn,
			f,
			addrCh,
			originalDstAddr)

		defer netRequest.CleanUp()

		var tConn *net.TCPConn
		for {
			dstAddr := <-addrCh
			if dstAddr == "" {
				f.Close()
				closeConn(logger, conn)
				close(connCh)
				return
			}

			tcpDstAddr, err := net.ResolveTCPAddr("tcp", dstAddr)
			if err != nil {
				logger.Warningf("Error while resolving tcp addr %s", originalDstAddr)
				connCh <- nil
				f.Close()
				closeConn(logger, conn)
				close(connCh)
				return
			}
			targetConn, err := net.DialTCP("tcp", nil, tcpDstAddr)
			if err != nil {
				logger.Warning(err.Error())
				connCh <- nil
				f.Close()
				closeConn(logger, conn)
				close(connCh)
				return
			}

			if tConn != nil {
				closeConn(logger, tConn)
			}
			tConn = targetConn
			connCh <- targetConn

			go func() {
				netHandler.HandleResponse(targetConn, conn, netRequest, isInBoundConn)
				closeConn(logger, targetConn)
			}()
		}
	} else {
		tcpDstAddr, err := net.ResolveTCPAddr("tcp", originalDstAddr)
		if err != nil {
			logger.Warningf("Error while resolving tcp addr %s", originalDstAddr)
			f.Close()
			closeConn(logger, conn)
			return
		}
		targetConn, err := net.DialTCP("tcp", nil, tcpDstAddr)
		if err != nil {
			logger.Warning(err.Error())
			f.Close()
			closeConn(logger, conn)
			return
		}

		go TcpCopyRequest(
			logger,
			conn,
			targetConn,
			nil,
			netRequest,
			netHandler,
			isInBoundConn,
			f,
			nil,
			originalDstAddr)

		go TcpCopyResponse(logger, targetConn, conn, netRequest, netHandler, isInBoundConn, f)
	}

	//ec.Remove(dstAddr)
}

func closeConn(logger *log.Logger, conn *net.TCPConn) {
	logger.Debug("Closing conn")
	// Important to close read operations
	// to avoid waiting for never ending read operation when client doesn't close connection
	conn.CloseRead()
	conn.CloseWrite()
	conn.Close()
	logger.Debug("Closed conn")
}
