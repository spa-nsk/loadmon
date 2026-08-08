package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type appConfig struct {
	DBFile    string `json:"DBFile"`
	DBFileAdd string `json:"DBFileAdd"`
}

func loadDBPaths(configPath string) (mainDB, addDB string, err error) {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", err
	}
	var cfg appConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", "", err
	}
	confDir := filepath.Dir(configPath)
	mainDB = resolveDBPath(confDir, cfg.DBFile)
	addDB = resolveDBPath(confDir, cfg.DBFileAdd)
	return mainDB, addDB, nil
}

func resolveDBPath(confDir, dbFile string) string {
	if dbFile == "" {
		return ""
	}
	if filepath.IsAbs(dbFile) {
		return dbFile
	}
	return filepath.Join(confDir, dbFile)
}
