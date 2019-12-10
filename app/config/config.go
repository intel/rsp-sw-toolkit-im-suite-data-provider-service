/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/configuration"
)

// ServiceConfig holds the service configuration.
type ServiceConfig struct {
	ServiceName            string
	LoggingLevel           string
	TelemetryEndpoint      string
	TelemetryDataStoreName string
	Port                   string

	// PipelinesDir is a directory containing pipeline configurations.
	PipelinesDir string
	// PipelineNames is a list of filenames to load from the pipelines directory.
	PipelineNames []string
	// PipelineTasks is a list filenames containing pipelines to use as custom
	// Task types. The files should be in the pipelines directory. They will be
	// loaded and added to the Plumber with a type name equal to the pipeline's
	// name; the output task should be named "output". Note that if multiple
	// pipelines have the same name, only the last loaded type will be used.
	CustomTaskTypes []string
	// TemplatesDir is a directory containing template namespace files.
	TemplatesDir string
	// SecretsPath is the path to docker secrets, usually /run/secrets.
	SecretsPath string
	// MQTTClients are files containing MQTT client configurations.
	MQTTClients []string
}

// AppConfig exports a package-level configuration object.
var AppConfig = ServiceConfig{}

// InitConfig loads package-level configuration.
func InitConfig() error {
	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	for _, required := range []struct {
		v    *string
		name string
	}{
		{v: &AppConfig.ServiceName, name: "serviceName"},
		{v: &AppConfig.LoggingLevel, name: "loggingLevel"},
		{v: &AppConfig.TelemetryEndpoint, name: "telemetryEndpoint"},
		{v: &AppConfig.TelemetryDataStoreName, name: "telemetryDataStoreName"},
		{v: &AppConfig.Port, name: "port"},
		{v: &AppConfig.PipelinesDir, name: "pipelinesDir"},
		{v: &AppConfig.TemplatesDir, name: "templatesDir"},
		{v: &AppConfig.SecretsPath, name: "secretsPath"},
	} {
		s, err := config.GetString(required.name)
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
		*required.v = s
	}

	AppConfig.PipelineNames, err = config.GetStringSlice("pipelineNames")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.CustomTaskTypes, err = config.GetStringSlice("customTaskTypes")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.MQTTClients, err = config.GetStringSlice("mqttClients")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	return nil
}
