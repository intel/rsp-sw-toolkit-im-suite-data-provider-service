//
// INTEL CONFIDENTIAL
// Copyright 2017 Intel Corporation.
//
// This software and the related documents are Intel copyrighted materials, and your use of them is governed
// by the express license under which they were provided to you (License). Unless the License provides otherwise,
// you may not use, modify, copy, publish, distribute, disclose or transmit this software or the related documents
// without Intel's prior written permission.
//
// This software and the related documents are provided as is, with no express or implied warranties, other than
// those that are expressly stated in the License.
//

package state

type MqttTopicMapping struct {
	UrlTemplate string   `json:"urlTemplate"`
	Topics      []string `json:"topics"`
}

func parseMqttTopicMapping(topicMap interface{}) MqttTopicMapping {
	tmInterface := topicMap.(map[string]interface{})
	topicInterface := tmInterface["topics"].([]interface{})

	tm := MqttTopicMapping{
		UrlTemplate: tmInterface["urlTemplate"].(string),
		Topics:      make([]string, len(topicInterface)),
	}

	for i, t := range topicInterface {
		tm.Topics[i] = t.(string)
	}
	return tm
}
