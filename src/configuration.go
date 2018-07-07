/**
go-build - Mulit-Project Build Utility by @Danw33
MIT License

Copyright 2017 - 2018 Daniel Wilson <hello@danw.io>

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// configuration - Configuration loading and parsing
package main

import (
	"encoding/json"
)

// Configuration defines the top-level structure used in the configuration file
type Configuration struct {
	Home     string          `json:"home"`
	Async    bool            `json:"async"`
	Log      LogConfig       `json:"log"`
	Plugins  []string        `json:"plugins"`
	Projects []ProjectConfig `json:"projects"`
}

// LogConfig defines the configuration available for the logger, and is utilised
// within the Configuration struct
type LogConfig struct {
	Level string `json:"level"`
}

// ProjectConfig defines the project-level configuration, and is utilised within
// the Configuration struct
type ProjectConfig struct {
	URL       string   `json:"url"`
	Path      string   `json:"path"`
	Artifacts string   `json:"artifacts"`
	Plugins   []string `json:"plugins"`
	Branches  []string `json:"branches"`
	Scripts   []string `json:"scripts"`
}

// parseConfig takes the given json string and uses json.Unmarshal to parse it
// using the Configuration struct
func parseConfig(cfg string) *Configuration {
	res := Configuration{}
	Log.Debug("Parsing Configuration using json.Unmarshal...\n")
	json.Unmarshal([]byte(cfg), &res)
	Log.Debugf("Loaded Configuration: %d Projects Configured.\n", len(res.Projects))
	return &res
}
