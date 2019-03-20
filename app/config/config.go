/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package config

import (
	"log"

	"github.com/pkg/errors"
	"github.impcloud.net/Responsive-Retail-Core/utilities/configuration"
)

type (
	ServiceConfig struct {
		ServiceName            string
		LoggingLevel           string
		TelemetryEndpoint      string
		TelemetryDataStoreName string
		Port                   string

		// BaseEndpoint is the template used for the base endpoint.
		BaseEndpoint string
		// MqttTopicMap keeps the key:value relationship between each endpoint
		// and the MQTT topic in which the result will be published
		MqttTopicMap map[string]interface{}
		// PollPeriodSecs is the wait time between checks for new manifests if
		// the last attempt succeeded.
		PollPeriodSecs uint
		// FailureRetryPeriodSecs is the amount of time to wait before polling
		// again if the last attempt failed.
		FailureRetryPeriodSecs uint
		// FailureRetries is the number of failures before retrying gives up.
		FailureRetries uint
		// ProxyThroughCloudConnector, if true, will enable the CC on downloads
		// and will require valid cloudConnectorEndpoint and oauth config settings
		ProxyThroughCloudConnector bool
		// CloudConnectorEndpoint is the endpoint of the cloud connector service,
		// through which downloads should be proxied (if enabled)
		CloudConnectorEndpoint string
		// OAuthEndpoint is the endpoint the cloud connector should use to get
		// an OAuth token
		OAuthEndpoint string
		// OAuthCredentials is the "username:password" pair (presumably) that
		// the cloud connector should use when getting an OAuth token
		OAuthCredentials string
		// EncryptGatewayConnection, if true, will connect to the MQTT broker
		// in the broker using TLS, or TCP if false.
		EncryptGatewayConnection bool
		// GatewayCredentialsPath is the path where the gateway credentials to
		// connect with the MQTT broker will be located
		GatewayCredentialsPath string
		// Gateway is the string value with the IP address of the MQTT broker
		// in the gateway
		Gateway string

		SecureMode, SkipCertVerify bool
	}
)

// AppConfig exports all config variables
var AppConfig ServiceConfig

// InitConfig loads application variables
func InitConfig() error {

	AppConfig = ServiceConfig{}

	var err error

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	log.Println(config)

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Set "debug" for development purposes. Nil for Production.
	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.Port, err = config.GetString("port")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.BaseEndpoint, err = config.GetString("baseEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.MqttTopicMap, err = config.GetNestedJSON("mqttTopicMapping")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	pollPeriod, err := config.GetInt("pollPeriodSecs")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if pollPeriod <= 0 {
		return errors.New("pollPeriodSecs must be >0")
	}
	AppConfig.PollPeriodSecs = uint(pollPeriod)

	retryPeriod, err := config.GetInt("failureRetryPeriodSecs")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if retryPeriod <= 0 {
		return errors.New("failureRetryPeriodSecs must be >0")
	}
	if retryPeriod > pollPeriod {
		return errors.New("failureRetryPeriodSecs must be <= pollPeriod")
	}
	AppConfig.FailureRetryPeriodSecs = uint(retryPeriod)

	retryLimit, err := config.GetInt("failureRetries")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if retryLimit <= 0 {
		return errors.New("failureRetryPeriodSecs must be >0")
	}
	AppConfig.FailureRetries = uint(retryLimit)

	AppConfig.ProxyThroughCloudConnector, err = config.GetBool("proxyThroughCloudConnector")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	if AppConfig.ProxyThroughCloudConnector {
		AppConfig.CloudConnectorEndpoint, err = config.GetString("cloudConnectorEndpoint")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
		AppConfig.OAuthEndpoint, err = config.GetString("oauthEndpoint")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
		AppConfig.OAuthCredentials, err = config.GetString("oauthCredentials")
		if err != nil {
			return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
		}
	}

	AppConfig.EncryptGatewayConnection, err = config.GetBool("encryptGatewayConnection")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.GatewayCredentialsPath, err = config.GetString("gatewayCredentialsPath")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.Gateway, err = config.GetString("gateway")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SecureMode, err = config.GetBool("secureMode")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.SkipCertVerify, err = config.GetBool("skipCertVerify")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	return nil
}
