package cli

import (
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/gitinspect"
	"github.com/romerramos/previously_on/internal/state"
)

type appContext struct {
	Config config.Config
	Store  *state.Store
	Repo   gitinspect.Repo
}

func loadContext() (appContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return appContext{}, err
	}

	repo, err := gitinspect.Discover()
	if err != nil {
		return appContext{}, err
	}

	store, err := state.NewStore(cfg.StateDir)
	if err != nil {
		return appContext{}, err
	}

	return appContext{Config: cfg, Store: store, Repo: repo}, nil
}
