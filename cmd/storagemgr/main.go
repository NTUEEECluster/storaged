package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configLoc := flag.String("config", "/etc/storaged/storagemgr.toml", "Location of config file")
	userName := flag.String("user", "", "Username to execute as")
	flag.Parse()

	var config Config
	_, err := toml.DecodeFile(*configLoc, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading configuration file:", err)
		os.Exit(1)
	}

	if *userName == "" {
		currentUser, err := user.Current()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error checking your username:", err)
			os.Exit(1)
		}
		userName = &currentUser.Username
	}
	p := tea.NewProgram(newStorageModel(config.StoragedAddr, *userName))
	_, err = p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "An unexpected error has occurred:", err)
		os.Exit(1)
	}
}

type Config struct {
	StoragedAddr string `json:"storaged_addr"`
}
