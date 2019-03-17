package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/opentracing/opentracing-go"
	//"github.com/panjf2000/ants"
	"github.com/patrickmn/go-cache"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	"github.com/Lookyan/netramesh/internal/config"
	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/log"
	"github.com/Lookyan/netramesh/pkg/transport"
)

func main() {
	//defer ants.Release()
	logger, err := log.Init("NETRA", os.Getenv(log.EnvNetraLoggerLevel), os.Stderr)
	if err != nil {
		log.Fatal(err.Error())
	}
	serviceName := flag.String("service-name", "", "service name for jaeger")
	flag.Parse()
	if *serviceName == "" {
		logger.Fatal("service-name flag should be set")
	}

	err = config.GlobalConfigFromENV(logger)
	if err != nil {
		logger.Fatal(err.Error())
	}

	go func() {
		// pprof
		logger.Error(
			http.ListenAndServe(
				fmt.Sprintf("0.0.0.0:%d", config.GetNetraConfig().PprofPort), nil))
	}()

	//go func() {
	//	for {
	//		logger.Debugf("Num of goroutines: %d", runtime.NumGoroutine())
	//		time.Sleep(5 * time.Second)
	//	}
	//}()

	os.Setenv("JAEGER_SERVICE_NAME", *serviceName)
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		logger.Fatalf("Could not parse Jaeger env vars: %s", err.Error())
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		logger.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	addr := fmt.Sprintf("0.0.0.0:%d", config.GetNetraConfig().Port)
	lAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logger.Fatal(err.Error())
	}

	ln, err := net.ListenTCP("tcp", lAddr)
	if err != nil {
		logger.Fatal(err.Error())
	}

	establishedCache := estabcache.NewEstablishedCache()
	//go func() {
	//	for {
	//		establishedCache.PrintConnections(logger)
	//		time.Sleep(5 * time.Second)
	//	}
	//}()

	tracingContextMapping := cache.New(
		config.GetNetraConfig().TracingContextExpiration,
		config.GetNetraConfig().TracingContextCleanupInterval,
	)

	//tcpCopyPool, _ := ants.NewPoolWithFunc(10000, func(i interface{})  {
	//	bucket := i.(*transport.TCPCopyBucket)
	//	transport.TcpCopy(
	//		logger,
	//		bucket.R,
	//		bucket.W,
	//		bucket.Initiator,
	//		bucket.NetRequest,
	//		bucket.NetHandler,
	//		bucket.IsInBoundConn,
	//		bucket.Done)
	//})

	//syncHandleConnection := func(c interface{}) {
	//	conn := c.(*net.TCPConn)
	//	transport.HandleConnection(logger, conn, establishedCache, tracingContextMapping, tcpCopyPool)
	//}

	//p, _ := ants.NewPoolWithFunc(10000, func(i interface{}) {
	//	syncHandleConnection(i)
	//})

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			logger.Warning(err.Error())
			continue
		}
		//p.Invoke(conn)
		go transport.HandleConnection(logger, conn, establishedCache, tracingContextMapping)
	}
}
