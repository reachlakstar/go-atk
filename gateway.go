package atk

import (
	"flag"
	"google.golang.org/grpc"
	"github.com/golang/glog"
	"net/http"
	"fmt"
	"time"
	"net"
	"context"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/go-log/log"
	"github.com/lakstap/go-atk/gateway"
	"strings"
	"reflect"
)

var (
	// the go.micro.srv.atk address
	port        = flag.String("port", ":8090", "go.micro.srv.atk.project address")
	endpoint    = flag.String("endpoint", "0.0.0.0:9090", "go.micro.srv.atk.project endpoint")
	network     = flag.String("network", "tcp", `one of "tcp" or "unix". Must be consistent to -network`)
	environment = flag.String("environment", "dev", `identify which environment application is running`)
	swaggerDir  = flag.String("swagger_dir", "proto/api", "path to the directory which contains swagger definitions")
)

// Endpoint describes a gRPC endpoint
type Endpoint struct {
	Network, Addr string
}

// Options is a set of options to be passed to Run
type ATKGateway struct {
	// Addr is the address to listen
	Addr string

	// Environment
	Env string

	// SwaggerDir is a path to a directory from which the server
	// serves swagger specs.
	SwaggerDir string

	// Mux is a list of options to be passed to the grpc-gateway multiplexer
	EndpointHandlers []EndpointHandler

	// Mux is a list of options to be passed to the grpc-gateway multiplexer
	Mux []gwruntime.ServeMuxOption
}

type EndpointHandlerOption func(*ATKGateway)
type EndpointHandler func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error

//type EndpointHandler func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error

func WithEndpointHandlerOption(handler EndpointHandler) EndpointHandlerOption {
	return func(gwOption *ATKGateway) {
		gwOption.EndpointHandlers = append(gwOption.EndpointHandlers, handler)
	}
}

// New ATK Gateway returns a new gateway with default values.
func NewATKGateway(opts ...EndpointHandlerOption) *ATKGateway {

	atkGateway := &ATKGateway{
		EndpointHandlers: make([]EndpointHandler, 0),
		Addr:             *port,
		SwaggerDir:       *swaggerDir,
		Env:              *environment,
	}

	for _, opt := range opts {
		opt(atkGateway)
	}
	return atkGateway
}

// Run starts a HTTP server and blocks while running if successful.
// The server will be shutdown when "ctx" is canceled.
func (gw *ATKGateway) RunGateway(ctx context.Context, options ...interface{}) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/swagger.json", gateway.ServeSwaggerJSON(gw.SwaggerDir))
	gwy, err := newGateway(ctx, gw.EndpointHandlers, gw.Mux)
	if err != nil {
		return err
	}
	if len(options) > 0 && options[0] == true {
		urls := strings.Split(reflect.ValueOf(options[1]).String(), ":")
		for _, url := range urls {
			mux.Handle("/"+url+"/", gateway.AuthMiddleware(ctx, gwy))
		}
	}

	//mux.Handle("/health/", gateway.DefaultAuthMiddleware(ctx, gwy))
	mux.Handle("/", gwy)

	gateway.SwaggerServer(mux)

	s := &http.Server{
		Addr:    gw.Addr,
		Handler: gateway.SetupGlobalMiddleware(mux),
	}

	go func() {
		<-ctx.Done()
		glog.Infof("Shutting down the http server")
		if err := s.Shutdown(context.Background()); err != nil {
			glog.Errorf("Failed to shutdown http server: %v", err)
		}
	}()
	isHTTPSEnabled := false;
	if len(options) > 0 && options[2] == true {
		isHTTPSEnabled = reflect.ValueOf(options[2]).Bool()
	}
	log.Logf("Server Started  listening at the address at %s is HTTPS Enable (%s)", gw.Addr, isHTTPSEnabled)
	if isHTTPSEnabled {
		if err := s.ListenAndServeTLS("/etc/secrets/server.crt", "/etc/secrets/server.key"); err != http.ErrServerClosed {
			glog.Errorf("Failed to listen and serve: %v", err)
			return err
		}
	} else {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			glog.Errorf("Failed to listen and serve: %v", err)
			return err
		}
	}
	return nil
}

// newGateway returns a  gateway server which translates HTTP into gRPC.
func newGateway(ctx context.Context, handlers []EndpointHandler, opts []gwruntime.ServeMuxOption) (http.Handler, error) {
	opts = append(opts, gwruntime.WithMetadata(gateway.ForwardAuthenticationMetadata))
	opts = append(opts, gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.JSONPb{OrigName: true, EmitDefaults: true}))
	mux := gwruntime.NewServeMux(opts...)
	dialopts := []grpc.DialOption{grpc.WithInsecure(), gateway.WithClientUnaryInterceptor(*environment)}

	endpoints := strings.Split(*endpoint, ",")

	for i, f := range handlers {
		if err := f(ctx, mux, endpoints[i], dialopts); err != nil {
			//if err := f(ctx, mux, conn); err != nil {
			fmt.Println("ERR: Failed to getting connect end point")
			return nil, err
		}
	}

	return mux, nil
}

func dial(ctx context.Context, network, addr string) (*grpc.ClientConn, error) {
	switch network {
	case "tcp":
		return dialTCP(ctx, addr)
	case "unix":
		return dialUnix(ctx, addr)
	default:
		return nil, fmt.Errorf("unsupported network type %q", network)
	}
}

// dialTCP creates a client connection via TCP.
// "addr" must be a valid TCP address with a port number.
func dialTCP(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, addr, grpc.WithInsecure())
}

// dialUnix creates a client connection via a unix domain socket.
// "addr" must be a valid path to the socket.
func dialUnix(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	d := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}
	return grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithDialer(d))
}
