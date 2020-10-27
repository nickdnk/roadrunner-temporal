package temporal

import (
	"github.com/spiral/endure/errors"
	"github.com/spiral/roadrunner/v2"
	"github.com/spiral/roadrunner/v2/plugins/config"
	rrt "github.com/temporalio/roadrunner-temporal"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

const ServiceName = "temporal"

type Config struct {
	Address    string
	Namespace  string
	Activities *roadrunner.Config
}

type Temporal interface {
	GetClient() (client.Client, error)
	GetConfig() Config
	CreateWorker(taskQueue string, options worker.Options) (worker.Worker, error)
}

// inherit roadrunner.rpc.Plugin interface
type Server struct {
	cfg    Config
	log    *zap.Logger
	client client.Client
}

// logger dep also
func (srv *Server) Init(cfg config.Provider, log *zap.Logger) error {
	srv.log = log
	return cfg.UnmarshalKey(ServiceName, &srv.cfg)
}

// GetConfig returns temporal configuration.
func (srv *Server) GetConfig() Config {
	return srv.cfg
}

// Serve starts temporal client.
func (srv *Server) Serve() chan error {
	errCh := make(chan error, 1)
	var err error

	srv.client, err = client.NewClient(client.Options{
		Logger:        &ZapAdapter{zl: srv.log},
		HostPort:      srv.cfg.Address,
		Namespace:     srv.cfg.Namespace,
		DataConverter: rrt.NewDataConverter(),
	})

	srv.log.Debug("Connected to temporal server", zap.String("Server", srv.cfg.Address))

	if err != nil {
		errCh <- errors.E(errors.Op("client connect"), err)
	}

	return errCh
}

// Stop stops temporal client connection.
func (srv *Server) Stop() error {
	if srv.client != nil {
		srv.client.Close()
	}

	return nil
}

// GetClient returns active client connection.
func (srv *Server) GetClient() (client.Client, error) {
	return srv.client, nil
}

// CreateWorker allocates new temporal worker on an active connection.
func (srv *Server) CreateWorker(tq string, options worker.Options) (worker.Worker, error) {
	if srv.client == nil {
		return nil, errors.E("unable to create worker, invalid temporal client")
	}

	return worker.New(srv.client, tq, options), nil
}

// Name of the service.
func (srv *Server) Name() string {
	return ServiceName
}

// RPCService returns associated rpc service.
func (srv *Server) RPCService() (interface{}, error) {
	c, err := srv.GetClient()
	if err != nil {
		return nil, err
	}

	return &rpc{client: c}, nil
}
