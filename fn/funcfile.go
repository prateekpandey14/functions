package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

var (
	validfn = [...]string{
		"func.yaml",
		"func.yml",
		"func.json",
	}

	errUnexpectedFileFormat = errors.New("unexpected file format for function file")
)

type funcfile struct {
	App            *string           `yaml:"app,omitempty",json:"app,omitempty"`
	Name           string            `yaml:"name,omitempty",json:"name,omitempty"`
	Version        string            `yaml:"version,omitempty",json:"version,omitempty"`
	Runtime        *string           `yaml:"runtime,omitempty",json:"runtime,omitempty"`
	Entrypoint     *string           `yaml:"entrypoint,omitempty",json:"entrypoint,omitempty"`
	Route          *string           `yaml:"route,omitempty",json:"route,omitempty"`
	Type           *string           `yaml:"type,omitempty",json:"type,omitempty"`
	Memory         *int64            `yaml:"memory,omitempty",json:"memory,omitempty"`
	Format         *string           `yaml:"format,omitempty",json:"format,omitempty"`
	MaxConcurrency *int              `yaml:"int,omitempty",json:"int,omitempty"`
	Config         map[string]string `yaml:"config,omitempty",json:"config,omitempty"`
	Build          []string          `yaml:"build,omitempty",json:"build,omitempty"`
}

func (ff *funcfile) FullName() string {
	fname := ff.Name
	if ff.Version != "" {
		fname = fmt.Sprintf("%s:%s", fname, ff.Version)
	}
	return fname
}

func (ff *funcfile) RuntimeTag() (runtime, tag string) {
	if ff.Runtime == nil {
		return "", ""
	}

	rt := *ff.Runtime
	tagpos := strings.Index(rt, ":")
	if tagpos == -1 {
		return rt, ""
	}

	return rt[:tagpos], rt[tagpos+1:]
}

func findFuncfile() (*funcfile, error) {
	for _, fn := range validfn {
		if exists(fn) {
			return parsefuncfile(fn)
		}
	}
	return nil, newNotFoundError("could not find function file")
}

func parsefuncfile(path string) (*funcfile, error) {
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return decodeFuncfileJSON(path)
	case ".yaml", ".yml":
		return decodeFuncfileYAML(path)
	}
	return nil, errUnexpectedFileFormat
}

func storefuncfile(path string, ff *funcfile) error {
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return encodeFuncfileJSON(path, ff)
	case ".yaml", ".yml":
		return encodeFuncfileYAML(path, ff)
	}
	return errUnexpectedFileFormat
}

func decodeFuncfileJSON(path string) (*funcfile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s for parsing. Error: %v", path, err)
	}
	ff := new(funcfile)
	err = json.NewDecoder(f).Decode(ff)
	return ff, err
}

func decodeFuncfileYAML(path string) (*funcfile, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s for parsing. Error: %v", path, err)
	}
	ff := new(funcfile)
	err = yaml.Unmarshal(b, ff)
	return ff, err
}

func encodeFuncfileJSON(path string, ff *funcfile) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open %s for encoding. Error: %v", path, err)
	}
	return json.NewEncoder(f).Encode(ff)
}

func encodeFuncfileYAML(path string, ff *funcfile) error {
	b, err := yaml.Marshal(ff)
	if err != nil {
		return fmt.Errorf("could not encode function file. Error: %v", err)
	}
	return ioutil.WriteFile(path, b, os.FileMode(0644))
}
