/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
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

package configuration

import (
	"fmt"
	"github.impcloud.net/Responsive-Retail-Core/utilities/consulApi"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"encoding/json"
)

type Configuration struct {
	isNestedConfig       bool
	parsedJson           map[string]interface{}
	sectionName          string
	configChangeCallback func([]ChangeDetails)
}

type ChangeType uint

const (
	Invalid ChangeType = iota
	Added
	Updated
	Deleted
)

type ChangeDetails struct {
	Name      string
	Value     interface{}
	Operation ChangeType
}

func NewSectionedConfiguration(sectionName string) (*Configuration, error) {
	config := Configuration{}
	config.sectionName = sectionName

	err := config.loadConfiguration()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func NewConfiguration() (*Configuration, error) {
	config := Configuration{}

	_, executablePath, _, ok := runtime.Caller(2)
	if ok {
		config.sectionName = path.Base(path.Dir(executablePath))
	}

	err := config.loadConfiguration()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (config *Configuration) SetConfigChangeCallback(callback func([]ChangeDetails)) {
	config.configChangeCallback = callback
}

func (config *Configuration) Load(path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &config.parsedJson)
	return err
}

func (config *Configuration) GetParsedJson() map[string]interface{} {
	return config.parsedJson
}

func (config *Configuration) GetNestedJSON(path string) (map[string]interface{}, error) {
	config.isNestedConfig = true
	if !config.pathExistsInConfigFile(path) {
		return nil, fmt.Errorf("%s not found", path)
	}

	item := config.getValue(path)
	value, ok := item.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert value for '%s' to a map[string]interface: Value='%v'", path, item)
	}
	config.isNestedConfig = false
	return value, nil

}

func (config *Configuration) GetNestedMapOfMapString(path string) (map[string]map[string]string, error) {
	mapOfInterface, err := config.GetNestedJSON(path)
	if err != nil {
		return nil, err
	}
	nestedKeyValue := make(map[string]map[string]string)
	for key, value := range mapOfInterface {
		switch value.(type) {
		case map[string]interface{}:
			nestedValueToString, err := interfaceToString(value.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			nestedKeyValue[key] = nestedValueToString
		default:
			return nil, fmt.Errorf("unexpected type found %s for value='%v' while conversion", reflect.TypeOf(value), value)
		}

	}
	return nestedKeyValue, nil
}

func interfaceToString(values map[string]interface{}) (map[string]string, error) {
	mapOfString := make(map[string]string)
	for key, value := range values {
		valType := reflect.TypeOf(value)
		fmt.Print(valType)
		switch value.(type) {
		case float64:
			mapOfString[key] = strconv.FormatFloat(value.(float64), 'f', -1, 64)
		case bool:
			mapOfString[key] = strconv.FormatBool(value.(bool))
		case string:
			mapOfString[key] = value.(string)
		default:
			return nil, fmt.Errorf("unexpected type found %s for value='%v' during conversion. currently accepts only float64,bool and string", reflect.TypeOf(value), value)
		}

	}
	return mapOfString, nil
}

func (config *Configuration) GetString(path string) (string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return "", fmt.Errorf("%s not found", path)
		}

		return value, nil
	}

	item := config.getValue(path)

	value, ok := item.(string)
	if !ok {
		return "", fmt.Errorf("unable to convert value for '%s' to a string: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetInt(path string) (int, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, intValue)
		}

		return intValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return int(value), nil
}

func (config *Configuration) GetFloat(path string) (float64, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, value)
		}

		return floatValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetBool(path string) (bool, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return false, fmt.Errorf("%s not found", path)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("unable to convert value for '%s' to a bool: Value='%v'", path, boolValue)
		}

		return boolValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(bool)
	if !ok {
		return false, fmt.Errorf("unable to convert value for '%s' to a bool: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetStringSlice(path string) ([]string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return nil, fmt.Errorf("%s not found", path)
		}

		value = strings.Replace(value, "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		slice := strings.Split(value, ",")
		var resultSlice []string
		for _, item := range slice {
			resultSlice = append(resultSlice, strings.Trim(item, " "))
		}

		return resultSlice, nil
	}

	item := config.getValue(path)
	slice := item.([]interface{})

	var stringSlice []string
	for _, sliceItem := range slice {
		value, ok := sliceItem.(string)
		if !ok {
			return nil, fmt.Errorf("unable to convert a value for '%s' to a string: Value='%v'", path, sliceItem)

		}
		stringSlice = append(stringSlice, value)
	}

	return stringSlice, nil
}

func (config *Configuration) getValue(path string) interface{} {
	if config.parsedJson == nil {
		return nil
	}

	if config.sectionName != "" {
		sectionedPath := fmt.Sprintf("%s.%s", config.sectionName, path)
		value := config.getValueFromJson(sectionedPath)
		if value != nil {
			return value
		}
	}

	value := config.getValueFromJson(path)
	return value
}

func (config *Configuration) getValueFromJson(path string) interface{} {
	pathNodes := strings.Split(path, ".")
	if len(pathNodes) == 0 {
		return nil
	}

	var ok bool
	var value interface{}
	jsonNodes := config.parsedJson
	for _, node := range pathNodes {
		if jsonNodes[node] == nil {
			return nil
		}

		item := jsonNodes[node]
		jsonNodes, ok = item.(map[string]interface{})
		if ok && !config.isNestedConfig {
			continue
		}

		if config.sectionName == node {
			continue
		}

		value = item
		break
	}

	return value
}

func (config *Configuration) pathExistsInConfigFile(path string) bool {
	if config.sectionName != "" {
		sectionPath := fmt.Sprintf("%s.%s", config.sectionName, path)
		if config.getValue(sectionPath) != nil {
			return true
		}
	}

	if config.getValue(path) != nil {
		return true
	}

	return false
}

func (config *Configuration) loadConfiguration() error {
	_, filename, _, ok := runtime.Caller(2)
	if !ok {
		log.Print("No caller information")
	}

	absolutePath := path.Join(path.Dir(filename), "configuration.json")

	// By default load local configuration file if it exists
	if _, err := os.Stat(absolutePath); err != nil {
		absolutePath, ok = os.LookupEnv("runtimeConfigPath")
		if !ok {
			absolutePath = "/run/secrets/configuration.json"
		}
		if _, err := os.Stat(absolutePath); err != nil {
			absolutePath = ""
		}
	}

	consulUrl, urlOk := os.LookupEnv("consulUrl")
	consulConfigKey, keyOk := os.LookupEnv("consulConfigKey")
	if urlOk && keyOk {
		return config.loadFromConsul(absolutePath, consulUrl, consulConfigKey)
	}

	log.Print("consulUrl and/or consulConfigKey environment variable not set, using local configuration file")

	if absolutePath != "" {
		err := config.Load(absolutePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (config *Configuration) loadFromConsul(configFilePath string, consulUrl string, consulConfigKey string) error {

	consul, clientErr := consulApi.NewClient(&consulApi.Config{Address: consulUrl})
	if clientErr != nil {
		return fmt.Errorf("not able to communicate with Consul service: %s", clientErr.Error())
	}

	keyValuePair, checkErr := checkAndUpdateFromLocal(consul, consulConfigKey, configFilePath)
	if checkErr != nil {
		return checkErr
	}

	if err := json.Unmarshal(keyValuePair.Value, &config.parsedJson); err != nil {
		return fmt.Errorf("error marshaling JSON configuration received from/pushed to Consul Service: %s", err.Error())
	}

	// Now that we know we are using Consul service, we need to create a watch on the configuration for changes.
	watcher, watcherErr := NewWatcher(consul, consulConfigKey)
	if watcherErr != nil {
		return fmt.Errorf("error creating watcher for chnages to value for %s: %s", consulConfigKey, watcherErr.Error())
	}

	if err := watcher.Start(config.processConfigurationChanged); err != nil {
		return fmt.Errorf("error starting watcher for chnages to value for %s: %s", consulConfigKey, err.Error())
	}

	return nil
}

func (config *Configuration) applyConfigurationJson(jsonBytes []byte) error {

	// Clear parsed JSON so start fresh since old deleted fields don't get removed.
	config.parsedJson = map[string]interface{}{}

	return json.Unmarshal(jsonBytes, &config.parsedJson)
}

func (config *Configuration) processConfigurationChanged(configurationJson []byte) {

	// Have to get these before applying new configuration JSON for comparing later
	previousGlobalSection, previousTargetSection := config.getGlobalAndTargetSections()

	// This saves the new configuration
	if err := config.applyConfigurationJson(configurationJson); err != nil {
		log.Printf("error marshaling JSON configuration received from change Consul watcher: %s", err.Error())
	}

	// if callback not set there is no need to continue the processing looking if anything changed.
	if config.configChangeCallback == nil {
		return
	}

	var changedList []ChangeDetails
	newGlobalSection, newTargetSection := config.getGlobalAndTargetSections()

	changedList = config.getChanges(changedList, previousGlobalSection, newGlobalSection, false)
	changedList = config.getChanges(changedList, previousTargetSection, newTargetSection, true)

	if len(changedList) > 0 {
		config.configChangeCallback(changedList)
	}
}

func (config *Configuration) getGlobalAndTargetSections() (map[string]interface{}, map[string]interface{}) {

	globalSection := make(map[string]interface{})
	targetSection := make(map[string]interface{})

	for configItemName, configItemValue := range config.parsedJson {
		configValueDetail := reflect.ValueOf(configItemValue)
		kind := configValueDetail.Kind()

		if kind == reflect.Map {
			if configItemName == config.sectionName {
				for _, key := range configValueDetail.MapKeys() {
					valueFromMap := configValueDetail.MapIndex(key)
					name := key.Interface().(string)
					value := valueFromMap.Elem().Interface()

					targetSection[name] = value
				}
			}
		} else {
			globalSection[configItemName] = configItemValue
		}
	}

	return globalSection, targetSection
}

func (config *Configuration) getChanges(changedList []ChangeDetails, previousSection map[string]interface{}, newSection map[string]interface{}, isTargetSection bool) []ChangeDetails {
	for itemName, itemValue := range previousSection {
		name := itemName
		if isTargetSection {
			name = config.sectionName + "." + itemName
		}

		if newSection[itemName] == nil {
			details := ChangeDetails{
				Name:      name,
				Value:     nil,
				Operation: Deleted,
			}
			changedList = append(changedList, details)
			continue
		}

		if itemValue != newSection[itemName] {
			details := ChangeDetails{
				Name:      name,
				Value:     newSection[itemName],
				Operation: Updated,
			}
			changedList = append(changedList, details)
		}
	}

	for itemName, itemValue := range newSection {
		name := itemName
		if isTargetSection {
			name = config.sectionName + "." + itemName
		}

		if previousSection[itemName] == nil {
			details := ChangeDetails{
				Name:      name,
				Value:     itemValue,
				Operation: Added,
			}
			changedList = append(changedList, details)
		}
	}

	return changedList
}

func checkAndUpdateFromLocal(consul *consulApi.Client, consulConfigKey string, configFilePath string) (*consulApi.KeyValuePair, error) {
	keyValuePair, err := consul.GetValue(consulConfigKey, nil)
	if err != nil {
		return nil, fmt.Errorf("error attempting to get '%s' value from Consul service: %s", consulConfigKey, err.Error())
	}

	if keyValuePair == nil {
		log.Printf("%s not found in Consul Service. Attempting to push local default configuration to Consul Service", consulConfigKey)

		// Load the local default configuration file in order to push it to Consul.
		fileBytes, readErr := ioutil.ReadFile(configFilePath)
		if readErr != nil {
			return nil, fmt.Errorf("error attempting to load default configuration inorder to push to Consul Service: %s", readErr.Error())
		}

		keyValuePair = &consulApi.KeyValuePair{
			Key:   consulConfigKey,
			Value: fileBytes,
		}

		if putErr := consul.PutValue(consulConfigKey, string(fileBytes)); putErr != nil {
			return nil, fmt.Errorf("error pushing default configuration to '%s' value in Consul Service: %s", consulConfigKey, putErr.Error())
		}
	}

	return keyValuePair, nil
}
