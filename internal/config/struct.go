package config

import (
	"sync"
	"time"
)

// Individual stack

type Stack struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description,omitempty"`
	Method          string            `yaml:"method,omitempty"`
	UrlPath         string            `yaml:"urlPath,omitempty"`
	WorkDir         string            `yaml:"workDir,omitempty"`
	ContinueOnError bool              `yaml:"continueOnError,omitempty"`
	DependsOn       []string          `yaml:"dependsOn,omitempty"`
	Vars            map[string]string `yaml:"vars,omitempty"`
	ExecutionMode   string            `yaml:"executionMode,omitempty"`
	Count           int               `yaml:"count,omitempty"`
	Shell           string            `yaml:"shell,omitempty"`
	ShellArg        string            `yaml:"shellArg,omitempty"`
	Cmds            []string          `yaml:"cmds,omitempty"`
	Output          string            `yaml:"output,omitempty"`
}

// Root configuration

type Config struct {
	Stacks []Stack `yaml:"stacks"`
}

// Registry for holding outputs

type Executor struct {
	Registry  *Registry
	Config    *Config
	SourceDir string
}

// Result of each stack

type Result struct {
	Name            string        `json:"name" yaml:"name"`
	Success         bool          `json:"success" yaml:"success"`
	Output          string        `json:"output" yaml:"output"`
	Error           string        `json:"error" yaml:"error"`
	Duration        time.Duration `json:"duration" yaml:"duration"`
	ContinueOnError bool          `json:"continueOnError" yaml:"continueOnError"`
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

// Variable substitution

type VariableSubstitution struct {
	Vars   map[string]string
	Result map[string]string
}
