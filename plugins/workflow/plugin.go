package workflow

import (
	"context"

	"github.com/spiral/roadrunner/v2"

	"github.com/spiral/errors"
	"github.com/spiral/roadrunner/v2/interfaces/log"
	"github.com/spiral/roadrunner/v2/interfaces/server"
	"github.com/spiral/roadrunner/v2/util"
	"github.com/temporalio/roadrunner-temporal/plugins/temporal"
	"go.uber.org/zap"
)

const (
	// PluginName defines public service name.
	PluginName = "workflows"

	// RRMode sets as RR_MODE env variable to let worker know about the mode to run.
	RRMode = "temporal/workflow"
)

// Plugin manages workflows and workers.
type Plugin struct {
	temporal temporal.Temporal
	events   util.EventsHandler
	server   server.Server
	log      log.Logger
	pool     *workflowPool
}

// logger dep also
func (svc *Plugin) Init(temporal temporal.Temporal, server server.Server, log log.Logger) error {
	svc.temporal = temporal
	svc.server = server
	svc.log = log
	return nil
}

// Serve starts workflow service.
func (svc *Plugin) Serve() chan error {
	errCh := make(chan error, 1)

	pool, err := NewWorkflowPool(context.Background(), svc.server)
	if err != nil {
		errCh <- errors.E(errors.Op("newWorkflowPool"), err)
		return errCh
	}

	// todo: proxy events

	err = pool.Start(context.Background(), svc.temporal)
	if err != nil {
		errCh <- errors.E(errors.Op("startWorkflowPool"), err)
		return errCh
	}

	svc.pool = pool

	var workflows []string
	for workflow, _ := range pool.workflows {
		workflows = append(workflows, workflow)
	}

	svc.log.Debug("Started workflow processing", zap.Any("workflows", workflows))

	return errCh
}

// Stop workflow service.
func (svc *Plugin) Stop() error {
	if svc.pool != nil {
		svc.pool.Destroy(context.Background())
	}

	return nil
}

// Name of the service.
func (svc *Plugin) Name() string {
	return PluginName
}

// Name of the service.
func (svc *Plugin) Workers() []roadrunner.WorkerBase {
	return []roadrunner.WorkerBase{svc.pool.worker}
}

// Reset resets underlying workflow pool with new copy.
func (svc *Plugin) Reset() error {
	svc.log.Debug("Reset workflow pool")

	pool, err := NewWorkflowPool(context.Background(), svc.server)
	if err != nil {
		return errors.E(errors.Op("newWorkflowPool"), err)
	}

	// todo: proxy events

	err = pool.Start(context.Background(), svc.temporal)
	if err != nil {
		return errors.E(errors.Op("startWorkflowPool"), err)
	}

	previous := svc.pool
	svc.pool = pool

	var workflows []string
	for workflow, _ := range pool.workflows {
		workflows = append(workflows, workflow)
	}

	svc.log.Debug("Started workflow processing", zap.Any("workflows", workflows))
	previous.Destroy(context.Background())

	return nil
}

// AddListener adds event listeners to the service.
func (svc *Plugin) AddListener(listener util.EventListener) {
	svc.events.AddListener(listener)
}
