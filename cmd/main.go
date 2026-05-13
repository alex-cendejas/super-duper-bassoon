package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/super-duper-bassoon/internal/adapters/alert"
	"github.com/super-duper-bassoon/internal/adapters/enforcement"
	httpserver "github.com/super-duper-bassoon/internal/adapters/http"
	"github.com/super-duper-bassoon/internal/adapters/messaging"
	"github.com/super-duper-bassoon/internal/adapters/repository"
	"github.com/super-duper-bassoon/internal/adapters/trigger"
	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/services"
)

func main() {
	logger := log.New(os.Stdout, "[server] ", log.LstdFlags|log.LUTC)
	cfg, err := LoadConfig()
	if err != nil {
		logger.Fatalf("config error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		logger.Fatalf("mkdir db: %v", err)
	}

	db, err := repository.OpenSQLite(cfg.DBPath)
	if err != nil {
		logger.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	registry := repository.NewRegistry(db)

	natsConn, err := nats.Connect(cfg.NatsURL, nats.MaxReconnects(-1), nats.ReconnectWait(time.Second))
	if err != nil {
		logger.Printf("nats connect failed (will continue without): %v", err)
		natsConn = nil
	}
	defer func() {
		if natsConn != nil {
			natsConn.Close()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventBus := messaging.NewInMemoryEventBus(logger)

	var dispatcher *messaging.NATSMessageDispatcher
	var resultCh <-chan []byte
	if natsConn != nil {
		dispatcher = messaging.NewNATSMessageDispatcher(natsConn, logger)
		ch, err := dispatcher.SubscribeToResults("result.>", 1024)
		if err != nil {
			logger.Printf("nats subscribe failed: %v", err)
		} else {
			resultCh = ch
		}
	}

	alerter := alert.NewStdoutAlertPublisher(logger)
	blocker := enforcement.NewInMemoryDispatchBlocker()
	banEnf := services.NewBanEnforcementService(registry.Ban(), alerter, blocker, logger)
	if err := banEnf.WarmCache(ctx); err != nil {
		logger.Printf("warm ban cache: %v", err)
	}
	dispatchFilter := services.NewDispatchFilterService(banEnf, logger)

	dispatchPort := pickDispatcher(dispatcher)
	dispatchCoord := services.NewDispatchCoordinationService(registry.Run(), dispatchPort, registry.Client(), dispatchFilter, logger)
	groupSvc := services.NewDynamicGroupingService(registry.Client())

	orchSvc := services.NewWorkflowOrchestrationService(registry.Workflow(), registry.Run(), registry.Client(), groupSvc, dispatchCoord, eventBus, logger)

	configRepo := services.NewDefaultConfigRepository(registry.Workflow(), cfg.HealthSuccessThreshold, cfg.HealthWindowSize)
	healthSvc := services.NewHealthMonitoringService(registry.Run(), registry.Result(), registry.Ban(), registry.Health(), registry.Workflow(), eventBus, configRepo, cfg.HealthWindowSize, logger)

	stateMgr := services.NewWorkflowStateManager(registry.Workflow())
	policyRepo := services.NewDefaultPolicyRepo(registry.Workflow(), cfg.CircuitBreakerSuccessThreshold, cfg.HealthWindowSize, cfg.CircuitBreakerCooldownMS)
	defaultPolicy := &domain.CircuitBreakerPolicy{
		SuccessThreshold: cfg.CircuitBreakerSuccessThreshold,
		EvaluationWindow: cfg.HealthWindowSize,
		CooldownPeriod:   time.Duration(cfg.CircuitBreakerCooldownMS) * time.Millisecond,
	}
	circuitSvc := services.NewCircuitBreakerService(registry.Health(), registry.Circuit(), policyRepo, registry.Workflow(), stateMgr, alerter, eventBus, defaultPolicy, logger)
	_ = eventBus.Subscribe("health.updated", circuitSvc.OnHealthUpdatedEvent)

	loopSvc := services.NewLoopDetectionService(registry.Run(), registry.Workflow(), registry.Ban(), banEnf, alerter, cfg.LoopThresholdMS, logger)

	resultDispatcher := services.NewResultMessageDispatcher(logger)
	clientRegSvc := services.NewClientRegistrationService(registry.Client(), logger)
	resultDispatcher.RegisterHandler(clientRegSvc)
	resultDispatcher.RegisterHandler(loopSvc)
	resultDispatcher.RegisterHandler(healthSvc)
	if resultCh != nil {
		go resultDispatcher.Start(ctx, resultCh)
	}

	// Subscribe to client registrations from super-client
	if natsConn != nil {
		natsConn.Subscribe("client.register", func(msg *nats.Msg) {
			var client domain.ClientMetadata
			if err := json.Unmarshal(msg.Data, &client); err != nil {
				logger.Printf("client.register: malformed message: %v", err)
				return
			}
			if client.ClientID == "" {
				return
			}
			if client.Labels == nil {
				client.Labels = map[string]string{}
			}
			if client.InnerState == nil {
				client.InnerState = map[string]interface{}{}
			}
			client.Active = true
			client.LastSeenAt = time.Now().UTC()
			if err := registry.Client().SaveClient(context.Background(), &client); err != nil {
				logger.Printf("client.register: save client %s: %v", client.ClientID, err)
			} else {
				logger.Printf("client registered: %s", client.ClientID)
			}
		})
	}

	cronEval := trigger.NewCronEvaluator()
	triggerSvc := services.NewTriggerCoordinationService(registry.Workflow(), orchSvc, cronEval, eventBus, cfg.TriggerCheckIntervalMS, logger)
	triggerSvc.Start(ctx)

	apiSvc := services.NewAPIHandlerService(registry.Workflow(), orchSvc, registry.Client(), registry.Run(), registry.Result(), registry.Ban(), banEnf, circuitSvc, healthSvc)

	natsConnected := func() bool { return natsConn != nil && natsConn.IsConnected() }
	dbHealthy := func() bool { return db.PingContext(ctx) == nil }

	router := httpserver.NewRouter(httpserver.Deps{
		API:           apiSvc,
		HealthRepo:    registry.Health(),
		Alerts:        alerter,
		NATSConnected: natsConnected,
		DBHealthy:     dbHealthy,
		Logger:        logger,
	})

	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	server := &nethttp.Server{
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.HTTPReadTimeoutMS) * time.Millisecond,
	}
	go func() {
		logger.Printf("listening on %s", addr)
		if err := server.Serve(listener); err != nil && err != nethttp.ErrServerClosed {
			logger.Printf("http serve: %v", err)
		}
	}()

	// Periodic health aggregation
	go runPeriodic(ctx, time.Duration(cfg.HealthAggregationIntervalMS)*time.Millisecond, func() {
		wfs, err := registry.Workflow().ListActiveWorkflows(ctx)
		if err != nil {
			return
		}
		seen := map[string]struct{}{}
		for _, w := range wfs {
			if _, ok := seen[w.WorkflowType]; ok {
				continue
			}
			seen[w.WorkflowType] = struct{}{}
			_, _ = healthSvc.AggregateWorkflowTypeHealth(ctx, w.WorkflowType)
		}
	})

	// Periodic circuit breaker evaluation
	go runPeriodic(ctx, time.Duration(cfg.CircuitBreakerCheckIntervalMS)*time.Millisecond, func() {
		_ = circuitSvc.EvaluateAllWorkflows(ctx)
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Printf("shutdown initiated")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutMS)*time.Millisecond)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	triggerSvc.Stop()
	resultDispatcher.Stop()
	cancel()
}

func runPeriodic(ctx context.Context, d time.Duration, f func()) {
	if d <= 0 {
		return
	}
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f()
		}
	}
}

var _ atomic.Bool

func pickDispatcher(n *messaging.NATSMessageDispatcher) interface {
	SendDispatch(ctx context.Context, d *domain.Dispatch) error
	SendBatchDispatches(ctx context.Context, list []*domain.Dispatch) error
} {
	if n != nil {
		return n
	}
	return messaging.NewChannelDispatcher()
}
