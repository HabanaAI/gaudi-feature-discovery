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
	"flag"
	"strings"
	"testing"
	"time"
)

func TestConfParse(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		envVars map[string]string
		conf    Conf
		expConf Conf
	}{
		{
			name: "no env and flags keeps defaults",
			conf: Conf{
				RunOnce:        false,
				SleepInterval:  10 * time.Second,
				OutputFilePath: "/tmp/dummy",
			},
			expConf: Conf{
				RunOnce:        false,
				SleepInterval:  10 * time.Second,
				OutputFilePath: "/tmp/dummy",
			},
		},
		{
			name: "env values override default",
			conf: Conf{
				RunOnce:        false,
				SleepInterval:  10 * time.Second,
				OutputFilePath: "/tmp/dummy",
			},
			envVars: map[string]string{
				"HFD_ONCE":        "true",
				"HFD_INTERVAL":    "5s",
				"HFD_OUTPUT_FILE": "/tmp/fromenv",
			},
			expConf: Conf{
				RunOnce:        true,
				SleepInterval:  5 * time.Second,
				OutputFilePath: "/tmp/fromenv",
			},
		},
		{
			name: "flags override defaults",
			conf: Conf{
				RunOnce:        false,
				SleepInterval:  10 * time.Second,
				OutputFilePath: "/tmp/dummy",
			},
			args: "--once --interval 180s --output-file /tmp/fromflags",
			expConf: Conf{
				RunOnce:        true,
				SleepInterval:  180 * time.Second,
				OutputFilePath: "/tmp/fromflags",
			},
		},
		{
			name: "flags override env vars when set",
			conf: Conf{
				RunOnce:        false,
				SleepInterval:  10 * time.Second,
				OutputFilePath: "/tmp/dummy",
			},
			envVars: map[string]string{
				"HFD_ONCE":        "true",
				"HFD_INTERVAL":    "5s",
				"HFD_OUTPUT_FILE": "/tmp/fromenv",
			},
			args: "--interval 180s --output-file /tmp/fromflags",
			expConf: Conf{
				RunOnce:        true,
				SleepInterval:  180 * time.Second,
				OutputFilePath: "/tmp/fromflags",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			f := flag.NewFlagSet("test", flag.ContinueOnError)

			err := parse(&tc.conf, f, strings.Fields(tc.args))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tc.expConf != tc.conf {
				t.Errorf("expected %v, got %v", tc.expConf, tc.conf)
			}
		})
	}
}
