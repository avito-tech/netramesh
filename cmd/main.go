package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/protocol"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

const SO_ORIGINAL_DST = 80

var bufferPool = sync.Pool{
	New: func() interface{} { return make([]byte, 0xfff) },
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

func tcpCopy(
	r io.ReadWriteCloser,
	w io.ReadWriteCloser,
	initiator bool,
	netRequest protocol.NetRequest,
	netHandler protocol.NetHandler,
	done chan bool) {
	pr, pw := io.Pipe()
	teeStreamReader := io.TeeReader(r, pw)

	if initiator {
		go netHandler.HandleRequest(pr, pw, netRequest)
	} else {
		go netHandler.HandleResponse(pr, pw, netRequest)
	}

	startD := time.Now()
	buf := bufferPool.Get().([]byte)
	written, err := io.CopyBuffer(w, teeStreamReader, buf)
	bufferPool.Put(buf)
	log.Printf("TCP connection Duration: %s", time.Since(startD).String())
	pw.Close()

	log.Printf("Written: %d", written)
	if err != nil {
		log.Printf("Err CopyBuffer: %s", err.Error())
	}
	done <- true
}

func handleConnection(conn *net.TCPConn, ec *estabcache.EstablishedCache) {
	if conn == nil {
		return
	}
	defer func() {
		log.Print("Closing src conn")
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

	dstAddr := fmt.Sprintf("%s:%d", ipv4, port)
	log.Printf("From: %s To: %s", conn.RemoteAddr(), conn.LocalAddr())
	log.Printf("Original destination :: %s", dstAddr)

	targetConn, err := net.Dial("tcp", dstAddr)
	if err != nil {
		log.Print(err.Error())
		return
	}
	defer func() {
		log.Print("Closing target conn")
		targetConn.Close()
		log.Print("Closed target conn")
	}()
	if noDelayConn, ok := targetConn.(setNoDelayer); ok {
		noDelayConn.SetNoDelay(true)
	}

	// determine protocol and choose logic
	p := protocol.Determine(dstAddr)
	log.Printf("Determined %s protocol", p)
	netRequest := protocol.GetNetRequest(p)
	netHandler := protocol.GetNetworkHandler(p)

	ec.Add(dstAddr)

	done := make(chan bool, 1)
	go tcpCopy(conn, targetConn, true, netRequest, netHandler, done)
	go tcpCopy(targetConn, conn, false, netRequest, netHandler, done)
	<-done

	log.Print("Finished")
	ec.Remove(dstAddr)
}

func main() {
	serviceName := flag.String("service-name", "", "service name for jaeger")
	flag.Parse()
	if *serviceName == "" {
		log.Fatal("service-name flag should be set")
	}

	go func() {
		// pprof
		log.Println(http.ListenAndServe("0.0.0.0:14957", nil))
	}()

	go func() {
		for {
			log.Printf("Num of goroutines: %d", runtime.NumGoroutine())
			time.Sleep(5 * time.Second)
		}
	}()

	os.Setenv("JAEGER_SERVICE_NAME", *serviceName)
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	addr := "0.0.0.0:14956"
	lAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err.Error())
	}

	ln, err := net.ListenTCP("tcp", lAddr)
	if err != nil {
		log.Fatal(err.Error())
	}

	establishedCache := estabcache.NewEstablishedCache()
	go func() {
		for {
			establishedCache.PrintConnections()
			time.Sleep(5 * time.Second)
		}
	}()

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Print(err.Error())
			continue
		}

		go handleConnection(conn, establishedCache)
	}

}
