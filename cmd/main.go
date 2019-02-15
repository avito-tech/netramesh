package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/patrickmn/go-cache"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	"github.com/Lookyan/netramesh/pkg/estabcache"
	"github.com/Lookyan/netramesh/pkg/protocol"
	"github.com/Lookyan/netramesh/pkg/transport"
)

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

	protocol.GlobalConfigFromENV()

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

	tracingContextMapping := cache.New(5*time.Second, 1*time.Second)

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Print(err.Error())
			continue
		}

		go transport.HandleConnection(conn, establishedCache, tracingContextMapping)
	}
}
