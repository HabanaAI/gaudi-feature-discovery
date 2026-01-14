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

/*
* TODO:
* Currently, we are not removing labels from the node
* when we directly updating it, when a label is missing.
* For example, if the driver is removed, FW labels are not available
* but they will still appear on the server.
 */

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/habana-internal/habana-feature-discovery/collector"
	"k8s.io/client-go/kubernetes"
)

// Version of the binary. This will be set using ldflags at compile time
var Version = "develop"

func main() {
	slog.Info("Started HFD")
	err := run(flag.CommandLine, os.Args[1:])
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	slog.Info("Finished HFD Successfully")
}

func run(f *flag.FlagSet, args []string) error {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	conf := Conf{
		RunOnce:        false,
		SleepInterval:  60 * time.Second,
		OutputFilePath: defaultFeatureFile,
		NFDEnabled:     false,
	}
	err := parse(&conf, f, args)
	if err != nil {
		return err
	}

	nodeName := os.Getenv("KUBERNETES_NODENAME")

	var kclient kubernetes.Interface
	if !conf.NFDEnabled {
		if nodeName == "" {
			return fmt.Errorf("env KUBERNETES_NODENAME is empty or missing. cannot start")
		}

		kclient, err = kubeClient()
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(conf.SleepInterval)
	defer ticker.Stop()
L:
	for {
		labels, err := collector.DefaultLabels()
		if err != nil {
			return fmt.Errorf("updating node labels: %w", err)
		}

		if conf.RunOnce {
			return createNFDLocalFile(labels, conf.OutputFilePath)
		}

		select {
		case <-ctx.Done():
			slog.Info("Context canceled by OS signal")
			break L
		case <-ticker.C:
			if conf.NFDEnabled {
				if err := createNFDLocalFile(labels, conf.OutputFilePath); err != nil {
					slog.Error(err.Error())
				}
			} else {
				if err := updateNodeLabels(ctx, kclient, nodeName, labels); err != nil {
					slog.Error(err.Error())
				}
			}
			slog.Info("Cycle completed", "next_cycle", conf.SleepInterval)
		}
	}

	// Cleanup
	err = removeOutputFile(conf.OutputFilePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing output file: %w", err)
	}

	return nil
}

func createNFDLocalFile(labels map[string]string, outputPath string) error {
	slog.Info("Creating local nfd file")

	var output bytes.Buffer
	for k, v := range labels {
		_, err := fmt.Fprintf(&output, "%s=%s\n", k, v)
		if err != nil {
			return fmt.Errorf("error formatting label %q: %w", k, err)
		}
	}

	slog.Info("Writing labels to output file", "path", outputPath)
	err := writeFileAtomically(outputPath, output.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing file %q: %w", outputPath, err)
	}

	return nil
}

func writeFileAtomically(outputFile string, contents []byte, perm os.FileMode) error {
	absFilePath, err := filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("retrieving absolute path of output file: %w", err)
	}
	absDirPath := filepath.Dir(absFilePath)

	tmpFile := path.Join(absDirPath, "hfd-temp")
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}

	_, err = f.Write(contents)
	if err != nil {
		err = f.Close()
		if err != nil {
			return fmt.Errorf("closing temp file after write error: %w", err)
		}
		return fmt.Errorf("writing data to temp file: %w", err)
	}

	// Explicit close before renaming
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	err = os.Rename(tmpFile, outputFile)
	if err != nil {
		return fmt.Errorf("moving temporary file to '%v': %w", outputFile, err)
	}

	// Write correct permission after file permission in case mask changed it.
	err = os.Chmod(outputFile, perm)
	if err != nil {
		return fmt.Errorf("setting permissions on '%v': %w", outputFile, err)
	}

	return nil
}

func removeOutputFile(path string) error {
	absFilePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to retrieve absolute path of output file: %v", err)
	}

	absDirPath := filepath.Dir(absFilePath)
	tmpDirPath := filepath.Join(absDirPath, "hfd-tmp")

	err = os.RemoveAll(tmpDirPath)
	if err != nil {
		return fmt.Errorf("failed to remove temporary output directory: %w", err)
	}

	err = os.Remove(absFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove output file: %w", err)
	}

	return nil
}
