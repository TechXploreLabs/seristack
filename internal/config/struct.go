package config

import (
	"sync"
	"time"
)

type Stack struct {
	Name            string            `yaml:"name"`
	WorkDir         string            `yaml:"workDir,omitempty"`
	ContinueOnError bool              `yaml:"continueOnError,omitempty"`
	DependsOn       []string          `yaml:"dependsOn,omitempty"`
	Vars            map[string]string `yaml:"vars,omitempty"`
	IsSerial        bool              `yaml:"isSerial,omitempty"`
	Count           int               `yaml:"count,omitempty"`
	Shell           string            `yaml:"shell,omitempty"`
	ShellArg        string            `yaml:"shellArg,omitempty"`
	Cmds            []string          `yaml:"cmds,omitempty"`
}

type Endpoint struct {
	Path      string `yaml:"path"`
	Method    string `yaml:"method"`
	Stackname string `yaml:"stackName"`
}

type Serverconfig struct {
	Port      string     `yaml:"port,omitempty"`
	Endpoints []Endpoint `yaml:"endpoint"`
}

type Config struct {
	Stacks []Stack       `yaml:"stacks"`
	Server *Serverconfig `yaml:"server,omitempty"`
}

type Executor struct {
	Registry  *Registry
	Config    *Config
	SourceDir string
}

type Result struct {
	Name     string
	Success  bool
	Output   string
	Error    error
	Duration time.Duration
}

// Shard represents a single shard in the registry
type Shard struct {
	Mu      sync.RWMutex
	Results map[string]*Result
	Vars    map[string]any
}

// Registry stores results with sharded locks for better concurrency
type Registry struct {
	Shards     []*Shard
	ShardCount uint32
}

type VariableSubstitution struct {
	Vars   map[string]string
	Result map[string]string
}
