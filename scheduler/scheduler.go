package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"

	"ops-worker/checker"
	"ops-worker/config"
	"ops-worker/healthcheck"
	"ops-worker/sender"
)

type registeredCheck struct {
	cfg config.CheckConfig
	chk checker.Checker
}

type Scheduler struct {
	cron        *cron.Cron
	sender      *sender.Sender
	healthcheck *healthcheck.HealthChecker
	checks      []registeredCheck
}

func New(s *sender.Sender, hc *healthcheck.HealthChecker) *Scheduler {
	// Use standard 5-field cron (min hour dom mon dow)
	c := cron.New()
	return &Scheduler{
		cron:        c,
		sender:      s,
		healthcheck: hc,
	}
}

func (s *Scheduler) AddCheck(cfg config.CheckConfig, chk checker.Checker) error {
	_, err := s.cron.AddFunc(cfg.Schedule, func() {
		ctx := context.Background()
		result := chk.Check(ctx)
		if err := s.sender.Send(ctx, result); err != nil {
			log.Printf("ERROR: failed to send result for check %q: %v", cfg.Name, err)
		}
	})
	if err != nil {
		return err
	}
	s.checks = append(s.checks, registeredCheck{cfg: cfg, chk: chk})
	return nil
}

// RunAll runs all registered checks immediately and sends results to the server.
func (s *Scheduler) RunAll(ctx context.Context) {
	for _, rc := range s.checks {
		result := rc.chk.Check(ctx)
		if err := s.sender.Send(ctx, result); err != nil {
			log.Printf("ERROR: failed to send initial result for check %q: %v", rc.cfg.Name, err)
		} else {
			log.Printf("INFO: initial check sent: %q (status: %s)", rc.cfg.Name, result.Status)
		}
	}
}

func (s *Scheduler) AddHealthcheck(schedule string) error {
	_, err := s.cron.AddFunc(schedule, func() {
		ctx := context.Background()
		if err := s.healthcheck.Send(ctx); err != nil {
			log.Printf("ERROR: failed to send healthcheck: %v", err)
		}
	})
	return err
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
