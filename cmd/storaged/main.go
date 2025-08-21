package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/NTUEEECluster/storaged"
)

func main() {
	var config Config
	configLoc := flag.String("config", "/etc/storaged/storaged.toml", "Location of config file")
	flag.Parse()

	_, err := toml.DecodeFile(*configLoc, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read :", err)
	}
	err = run(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
	}
}

type Config struct {
	ListenAddr        string                           `toml:"listen_addr"`
	AllowedEncodeHost string                           `toml:"allowed_encode_host"`
	ProjectDir        string                           `toml:"project_dir"`
	TierDir           map[string]string                `toml:"tier_dir"`
	Allocations       map[string][]storaged.Allocation `toml:"allocations"`
}

func run(cfg Config) error {
	cephFS, err := storaged.NewCephFS()
	if err != nil {
		return fmt.Errorf("error initializing CephFS: %w", err)
	}
	cfg.ProjectDir = strings.TrimPrefix(cfg.ProjectDir, "/")
	projectDir, err := storaged.SubFS(cephFS, cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("error finding project directory: %w", err)
	}
	tiers := make(map[string]storaged.QuotaFS)
	for tierName, tierDir := range cfg.TierDir {
		tierDir = strings.TrimPrefix(tierDir, "/")
		tierFS, err := storaged.SubFS(cephFS, tierDir)
		if err != nil {
			return fmt.Errorf("error finding tier directory %q: %w", tierDir, err)
		}
		tiers[tierName] = tierFS
	}
	_, allowedEncodeHost, err := net.ParseCIDR(cfg.AllowedEncodeHost)
	if err != nil {
		return fmt.Errorf("error parsing allowed encoding host: %w", err)
	}
	srv := storaged.NewServer(storaged.ServerConfig{
		AllowedEncodeHost: allowedEncodeHost,
		ProjectFS:         projectDir,
		Tiers:             tiers,
		Allocations:       cfg.Allocations,
	})
	err = srv.Listen(cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	return nil
}
