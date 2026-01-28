/*
Copyright 2026 The ingress-to-gateway Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	kubeconfig string
	namespace  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "ingress-to-gateway",
	Short: "The complete Ingress-NGINX to Gateway API migration tool",
	Long: `ingress-to-gateway is a comprehensive tool for migrating from Ingress-NGINX
to Kubernetes Gateway API. It provides deep analysis, conversion, and validation
capabilities with support for 17+ NGINX Ingress annotations.

Features:
  • Comprehensive audit of Ingress resources
  • Smart conversion to HTTPRoute with multiple split strategies
  • Advanced annotation support (17+ annotations)
  • Validation and readiness checks
  • Batch conversion with intelligent grouping
  • Detailed migration reports

Example usage:
  # Audit all Ingress resources
  ingress-to-gateway audit --all-namespaces

  # Convert a single Ingress
  ingress-to-gateway convert my-ingress -n default

  # Batch convert with per-host strategy
  ingress-to-gateway batch --split-mode=per-host

  # Validate HTTPRoute
  ingress-to-gateway validate httproute.yaml`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ingress-to-gateway.yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace")

	// Bind flags to viper
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ingress-to-gateway")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
