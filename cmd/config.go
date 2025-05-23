// Copyright 2024 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package cmd

import (
	"corteca/internal/configuration"
	"corteca/internal/tui"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read/write configuration values",
	Long:  "Read/write configuration values\nIf --global flag is not enabled application's corteca.yaml is affected",
	Example: `#Add (append) a configuration value
corteca config add

#Read a configuration value
corteca config get

#Set a configuration value
corteca config set`,
	Args: cobra.MaximumNArgs(1),
}

var getCmd = &cobra.Command{
	Use:   "get key",
	Short: "Read a configuration value",
	Long:  "Read a configuration value from global or application's corteca.yaml file",
	Example: `#Read build configuration value 'outputType'
corteca config get build.options.outputType

#Read beacon's configuration values
corteca config get devices.beacon`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: validConfigArgsFunc,
	Run: func(cmd *cobra.Command, args []string) {
		key := ""
		if len(args) > 0 {
			key = args[0]
		}
		doGetConfigValue(key)
	},
}

var setCmd = &cobra.Command{
	Use:   "set key value",
	Short: "Set a configuration value",
	Long:  "Set a configuration value to global or application's corteca.yaml file",
	Example: `#Set build configuration value 'outputType'
corteca config set build.options.outputType=oci

#Set beacon's configuration value 'addr'
corteca config set devices.beacon.addr ssh://root:passwd@192.168.67.5

#Set global configuration 'bool' value for cross compilation 'enabled'
corteca config --global set build.crossCompile.enabled false

#Set global configuration 'array' value for cross compilation 'args'
corteca config --global set build.crossCompile.args "["--reset", "-p", "yes"]"`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: validConfigArgsFunc,
	Run:               func(cmd *cobra.Command, args []string) { doSetConfigValue(args[0], args[1], false) },
}

var addCmd = &cobra.Command{
	Use:   "add key value",
	Short: "Add (append) a configuration value",
	Long:  "Add (append) a configuration value to global or application's corteca.yaml file",
	Example: `#Add to devices a device named 'beacon6' with its configuration fields
corteca config add devices "{beacon6: {addr: ssh://root:passwd@192.168.67.5, password2: beaconpass}}"`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: validConfigArgsFunc,
	Run:               func(cmd *cobra.Command, args []string) { doSetConfigValue(args[0], args[1], true) },
}

func init() {
	setCmd.PersistentFlags().BoolVar(&noRegen, "no-regen", false, "Skip regeneration of templates")
	addCmd.PersistentFlags().BoolVar(&noRegen, "no-regen", false, "Skip regeneration of templates")
	configCmd.PersistentFlags().BoolVar(&skipLocalConfig, "global", false, "Affect global config & ignore any project-local configuration")
	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(addCmd)
	rootCmd.AddCommand(configCmd)
}

func doGetConfigValue(key string) {
	var (
		field any
		err   error
	)

	if skipLocalConfig {
		field, err = configuration.ReadField(configGlobal, key)
	} else {
		field, err = configuration.ReadField(config, key)
	}
	assertOperation("retrieving config value", err)
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(configuration.INDENTATION)
	enc.Encode(field)
}

func doSetConfigValue(key, value string, append bool) {
	if skipLocalConfig {
		assertOperation("writing configuration value", configGlobal.WriteField(key, value, append))
		// TODO: validate configuration settings
		assertOperation("writing configuration file", configGlobal.WriteConfiguration(userConfigRoot, &configSystem))
	} else {
		if projectRoot == "" {
			tui.LogError("Called outside of project scope; refusing to modify global configuration unless '--global' is explicitly specified.")
			os.Exit(1)
		}
		assertOperation("writing configuration value", config.WriteField(key, value, append))
		// TODO: validate configuration settings
		assertOperation("validating application settings", validateAppSettings())
		assertOperation("writing configuration file", config.WriteConfiguration(projectRoot, &configGlobal))
		if !noRegen {
			requireProjectContext()
			doRegenTemplates(projectRoot, "")
		}
	}
}

func validConfigArgsFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		filteredKeys := config.GetSuggestions(toComplete)
		return filteredKeys, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
