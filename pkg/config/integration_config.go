// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017 Datadog, Inc.

package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/spf13/viper"
)

const (
	LOGS_RULES       = "LogsRules"
	TCP_TYPE         = "tcp"
	UDP_TYPE         = "udp"
	FILE_TYPE        = "file"
	EXCLUDE_AT_MATCH = "exclude_at_match"
	MASK_SEQUENCES   = "mask_sequences"
)

// LogsProcessingRule defines an exclusion or a masking rule to
// be applied on log lines
type LogsProcessingRule struct {
	Type                    string
	Name                    string
	ReplacePlaceholder      string `mapstructure:"replace_placeholder"`
	Pattern                 string
	Reg                     *regexp.Regexp
	ReplacePlaceholderBytes []byte
}

// IntegrationConfigLogSource represents a log source config, which can be for instance
// a file to tail or a port to listen to
type IntegrationConfigLogSource struct {
	Type string

	Port int    // Network
	Path string // File

	Service         string
	Logset          string
	Source          string
	SourceCategory  string
	Tags            string
	TagsPayload     []byte
	ProcessingRules []LogsProcessingRule `mapstructure:"log_processing_rules"`
}

// IntegrationConfig represents a dd agent config, which includes infra and logs parts
type IntegrationConfig struct {
	Logs []IntegrationConfigLogSource
}

// GetLogsSources returns a list of integration sources
func GetLogsSources() []*IntegrationConfigLogSource {
	return getLogsSources(LogsAgent)
}

func getLogsSources(config *viper.Viper) []*IntegrationConfigLogSource {
	return config.Get(LOGS_RULES).([]*IntegrationConfigLogSource)
}

// BuildLogsAgentIntegrationsConfigs looks for all yml configs in the ddconfdPath directory,
// and initializes the LogsAgent integrations configs
func BuildLogsAgentIntegrationsConfigs(ddconfdPath string) error {
	return buildLogsAgentIntegrationsConfig(LogsAgent, ddconfdPath)
}

func buildLogsAgentIntegrationsConfig(config *viper.Viper, ddconfdPath string) error {

	integrationConfigYamlFiles := availableIntegrationConfigs(ddconfdPath)
	logsSourceConfigs := []*IntegrationConfigLogSource{}

	for _, file := range integrationConfigYamlFiles {
		var integrationConfig IntegrationConfig
		var viperCfg = viper.New()
		viperCfg.SetConfigName(file)
		viperCfg.AddConfigPath(ddconfdPath)
		err := viperCfg.ReadInConfig()
		if err != nil {
			return err
		}
		err = viperCfg.Unmarshal(&integrationConfig)
		if err != nil {
			return err
		}

		for _, logSourceConfigIterator := range integrationConfig.Logs {
			logSourceConfig := logSourceConfigIterator
			err = validateSource(logSourceConfig)
			if err != nil {
				return err
			}

			rules, err := validateProcessingRules(logSourceConfig.ProcessingRules)
			if err != nil {
				return err
			}
			logSourceConfig.ProcessingRules = rules

			logSourceConfig.TagsPayload = buildTagsPayload(logSourceConfig.Tags, logSourceConfig.Source, logSourceConfig.SourceCategory)

			logsSourceConfigs = append(logsSourceConfigs, &logSourceConfig)
		}
	}
	config.Set(LOGS_RULES, logsSourceConfigs)
	return nil
}

// availableIntegrationConfigs lists yaml files in ddconfdPath
func availableIntegrationConfigs(ddconfdPath string) []string {
	var integrationConfigFiles []string
	files, _ := ioutil.ReadDir(ddconfdPath)
	for _, f := range files {
		var filename = f.Name()
		var extension = filepath.Ext(filename)
		var name = filename[0 : len(filename)-len(extension)]
		if extension == ".yaml" && name != DeprecatedConfig {
			integrationConfigFiles = append(integrationConfigFiles, name)
		}
	}
	return integrationConfigFiles
}

func validateSource(config IntegrationConfigLogSource) error {

	switch config.Type {
	case FILE_TYPE,
		TCP_TYPE,
		UDP_TYPE:
	default:
		return fmt.Errorf("A source must have a valid type (got %s)", config.Type)
	}

	if config.Type == FILE_TYPE && config.Path == "" {
		return fmt.Errorf("A file source must have a path")
	}

	if config.Type == TCP_TYPE && config.Port == 0 {
		return fmt.Errorf("A tcp source must have a port")
	}

	if config.Type == UDP_TYPE && config.Port == 0 {
		return fmt.Errorf("A udp source must have a port")
	}

	return nil
}

// validateProcessingRules checks the rules and raises errors if one is misconfigured
func validateProcessingRules(rules []LogsProcessingRule) ([]LogsProcessingRule, error) {
	for i, rule := range rules {
		if rule.Name == "" {
			return nil, fmt.Errorf("LogsAgent misconfigured: all log processing rules need a name")
		}
		switch rule.Type {
		case EXCLUDE_AT_MATCH:
			rules[i].Reg = regexp.MustCompile(rule.Pattern)
		case MASK_SEQUENCES:
			rules[i].Reg = regexp.MustCompile(rule.Pattern)
			rules[i].ReplacePlaceholderBytes = []byte(rule.ReplacePlaceholder)
		default:
			if rule.Type == "" {
				return nil, fmt.Errorf("LogsAgent misconfigured: type must be set for log processing rule `%s`", rule.Name)
			} else {
				return nil, fmt.Errorf("LogsAgent misconfigured: type %s is unsupported for log processing rule `%s`", rule.Type, rule.Name)
			}
		}
	}
	return rules, nil
}

// Given a list of tags, buildTagsPayload generates the bytes array that will be inserted
// into messages
func buildTagsPayload(configTags, source, sourceCategory string) []byte {

	tagsPayload := []byte{}
	if source != "" {
		tagsPayload = append(tagsPayload, []byte("[dd ddsource=\"")...)
		tagsPayload = append(tagsPayload, []byte(source)...)
		tagsPayload = append(tagsPayload, []byte("\"]")...)
	}

	if sourceCategory != "" {
		tagsPayload = append(tagsPayload, []byte("[dd ddsourcecategory=\"")...)
		tagsPayload = append(tagsPayload, []byte(sourceCategory)...)
		tagsPayload = append(tagsPayload, []byte("\"]")...)
	}

	if configTags != "" {
		tagsPayload = append(tagsPayload, []byte("[dd ddtags=\"")...)
		tagsPayload = append(tagsPayload, []byte(configTags)...)
		tagsPayload = append(tagsPayload, []byte("\"]")...)
	}

	if len(tagsPayload) == 0 {
		return []byte{'-'}
	}

	return tagsPayload
}
