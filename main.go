package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ops-worker/checker"
	"ops-worker/config"
	"ops-worker/healthcheck"
	"ops-worker/scheduler"
	"ops-worker/sender"
	"ops-worker/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	service := flag.Bool("service", false, "run as background service (requires config file)")
	send := flag.Bool("send", false, "run a check once and send result to server (requires config file)")
	configPath := flag.String("config", "/etc/ops-worker/config.yaml", "path to config file")
	checksPath := flag.String("checks", "", "path to checks file (service mode, overrides config)")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("ops-worker %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}

	if *service {
		runService(*configPath, *checksPath)
		return
	}

	if *send {
		runSend(*configPath, flag.Args())
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	chk, err := buildOnceChecker(args[0], args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := chk.Check(context.Background())
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func buildOnceChecker(typeName string, args []string) (checker.Checker, error) {
	switch typeName {
	case "cpu":
		return checker.NewCPUChecker("cpu"), nil
	case "memory":
		return checker.NewMemoryChecker("memory"), nil
	case "disk":
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		return checker.NewDiskChecker("disk", map[string]interface{}{"path": path}), nil
	case "process":
		if len(args) == 0 {
			return nil, fmt.Errorf("process: process name required\n  usage: ops-worker process <name>")
		}
		return checker.NewProcessChecker("process", map[string]interface{}{"process_name": args[0]}), nil
	case "docker":
		if len(args) == 0 {
			return nil, fmt.Errorf("docker: container name required\n  usage: ops-worker docker <container>")
		}
		return checker.NewDockerChecker("docker", map[string]interface{}{"container_name": args[0]}), nil
	case "external":
		if len(args) == 0 {
			return nil, fmt.Errorf("external: command required\n  usage: ops-worker external <command> [args...]")
		}
		return checker.NewExternalChecker("external", map[string]interface{}{
			"command": args[0],
			"args":    args[1:],
		}), nil
	default:
		return nil, fmt.Errorf("unknown check type %q", typeName)
	}
}

func runSend(configPath string, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: check type required\n")
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	chk, err := buildOnceChecker(args[0], args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result := chk.Check(ctx)

	sndr := sender.New(cfg.ReportURL(), cfg.Server.Password, cfg.Server.Timeout)
	if err := sndr.Send(ctx, result); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to send result: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("sent: %s (status: %s)\n", result.Name, result.Status)
}

func runService(configPath, checksPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("FATAL: failed to load config: %v", err)
	}

	checksFile := cfg.ChecksFile
	if checksPath != "" {
		checksFile = checksPath
	}

	checksCfg, err := config.LoadChecks(checksFile)
	if err != nil {
		log.Fatalf("FATAL: failed to load checks: %v", err)
	}

	sndr := sender.New(cfg.ReportURL(), cfg.Server.Password, cfg.Server.Timeout)
	hc := healthcheck.New(cfg.HealthURL(), cfg.Server.Password, cfg.Server.Timeout)
	sched := scheduler.New(sndr, hc)

	for _, checkCfg := range checksCfg.Checks {
		chk, err := buildChecker(checkCfg)
		if err != nil {
			log.Printf("WARN: skipping check %q: %v", checkCfg.Name, err)
			continue
		}
		if err := sched.AddCheck(checkCfg, chk); err != nil {
			log.Printf("WARN: failed to schedule check %q: %v", checkCfg.Name, err)
		} else {
			log.Printf("INFO: scheduled check %q (type: %s, schedule: %s)", checkCfg.Name, checkCfg.Type, checkCfg.Schedule)
		}
	}

	if cfg.Healthcheck.Schedule != "" {
		if err := sched.AddHealthcheck(cfg.Healthcheck.Schedule); err != nil {
			log.Printf("WARN: failed to schedule healthcheck: %v", err)
		} else {
			log.Printf("INFO: scheduled healthcheck (schedule: %s)", cfg.Healthcheck.Schedule)
		}
	}

	sched.Start()
	log.Printf("INFO: ops-worker %s started", version.Version)

	log.Println("INFO: running initial check for all items...")
	sched.RunAll(context.Background())
	log.Println("INFO: initial checks completed")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("INFO: shutting down...")
	sched.Stop()
	log.Println("INFO: ops-worker stopped")
}

func buildChecker(cfg config.CheckConfig) (checker.Checker, error) {
	opts := cfg.Options
	if opts == nil {
		opts = map[string]interface{}{}
	}

	switch cfg.Type {
	case "cpu":
		return checker.NewCPUChecker(cfg.Name), nil
	case "disk":
		return checker.NewDiskChecker(cfg.Name, opts), nil
	case "memory":
		return checker.NewMemoryChecker(cfg.Name), nil
	case "process":
		return checker.NewProcessChecker(cfg.Name, opts), nil
	case "docker":
		return checker.NewDockerChecker(cfg.Name, opts), nil
	case "external":
		return checker.NewExternalChecker(cfg.Name, opts), nil
	default:
		return nil, fmt.Errorf("unknown checker type: %s", cfg.Type)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ops-worker <check-type> [args]       run a check and print result to stdout
  ops-worker --send <check-type> [args] run a check once and send result to server
  ops-worker --service [flags]         run as background service

Check types:
  cpu                       CPU usage
  memory                    memory usage
  disk [path]               disk usage (default path: /)
  process <name>            process running check
  docker <container>        docker container check
  external <command> [args] run external command as check

Send/Service mode flags:
  --config PATH   config file (default: /etc/ops-worker/config.yaml)
  --checks PATH   checks file (overrides checks_file in config, service mode only)

Other flags:
  --version   print version and exit

Examples:
  ops-worker cpu
  ops-worker disk /var/log
  ops-worker process nginx
  ops-worker docker my-app
  ops-worker external /usr/local/bin/my-check.sh
  ops-worker --send cpu
  ops-worker --send --config /etc/ops-worker/config.yaml disk /var/log
  ops-worker --service --config /etc/ops-worker/config.yaml
`)
}
