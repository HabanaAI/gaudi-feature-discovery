/*
 * Copyright (c) 2022, HabanaLabs Ltd.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

var defaultFeatureFile = "/etc/kubernetes/node-feature-discovery/features.d/hfd"

var usage = fmt.Sprintf(`
Usage:
  hfd [--once | --interval=<seconds>] [--output-file=<file> | -o <file>] [--nfd]
  hfd -h | --help
  hfd --version

Options:
  -h --help                       Show this help message and exit.
  --version                       Display version and exit.
  -O, --once                      Run once and quit.
  -i,--interval=<seconds>         Time to sleep between labeling [Default: 60s].
  --nfd                           Enable the application as NFS local plugin.
  -o, --output-file PATH          Path to output file. [Default: %s]`, defaultFeatureFile)

// Conf : Type to represent options
type Conf struct {
	OutputFilePath string
	SleepInterval  time.Duration
	RunOnce        bool
	NFDEnabled     bool
}

func parse(c *Conf, s *flag.FlagSet, args []string) error {
	s.Usage = func() {
		_, err := fmt.Fprintln(os.Stderr, usage)
		if err != nil {
			slog.Error("Error printing usage", "error", err)
		}
	}

	if err := loadEnv(c); err != nil {
		return err
	}

	var (
		labelOnce     bool
		sleepInterval time.Duration
		outputFile    string
		nfdEnabled    bool
	)

	s.BoolVar(&labelOnce, "once", false, "Label once and exit")
	s.BoolVar(&labelOnce, "O", false, "Label once and exit")
	s.DurationVar(&sleepInterval, "interval", 0, "Time to sleep between labeling")
	s.DurationVar(&sleepInterval, "i", 0, "Time to sleep between labeling")
	s.StringVar(&outputFile, "output-file", "", "Output file path")
	s.StringVar(&outputFile, "o", "", "Output file path")
	s.BoolVar(&nfdEnabled, "nfd", false, "Enable the app as NFD local plugin")

	if err := s.Parse(args); err != nil {
		flag.Usage()
		return fmt.Errorf("parsing flags: %w", err)
	}

	if labelOnce && sleepInterval != 0 {
		slog.Warn("WARN: Ignoring sleep interval, running once")
	}

	// Take values from flags if configured
	c.RunOnce = cmp.Or(labelOnce, c.RunOnce)
	c.SleepInterval = cmp.Or(sleepInterval, c.SleepInterval)
	c.OutputFilePath = cmp.Or(outputFile, c.OutputFilePath)
	c.NFDEnabled = nfdEnabled

	return nil
}

func loadEnv(c *Conf) error {
	if val, ok := os.LookupEnv("HFD_ONCE"); ok && strings.EqualFold(val, "true") {
		c.RunOnce = true
	}
	if interval, ok := os.LookupEnv("HFD_INTERVAL"); ok {
		var err error
		c.SleepInterval, err = time.ParseDuration(interval)
		if err != nil {
			return fmt.Errorf("invalid value from env for interval option: %w", err)
		}
	}
	if outputFilePathTmp, ok := os.LookupEnv("HFD_OUTPUT_FILE"); ok {
		c.OutputFilePath = outputFilePathTmp
	}
	return nil
}
