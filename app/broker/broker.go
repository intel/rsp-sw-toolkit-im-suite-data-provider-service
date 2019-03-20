package broker

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	metrics "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
)

const (
	connectTimeout = 30 * time.Second
)

const (
	ProductItemTopic = "rfid/gw/productmasterdata"
	ProductItemType  = "productmasterdata"
)

var (
	ErrTimeout = errors.New("timeout")
)

//NewMosquittoClient instantiates a Mosquitto provider with the selected options
//in the providerOptions parameter
func NewMosquittoClient(providerOptions *MosquittoProviderOptions) (provider *MosquittoProvider) {
	// if caller did not already specify the connection scheme, and they want security, add it for them
	if !strings.Contains(providerOptions.Gateway, "://") {
		if providerOptions.EncryptGatewayConnection {
			providerOptions.Gateway = "tls://" + providerOptions.Gateway
		} else {
			providerOptions.Gateway = "tcp://" + providerOptions.Gateway
		}
	}

	providerOptions.MttqClient = mqtt.NewClient(mqtt.NewClientOptions().
		AddBroker(providerOptions.Gateway).
		SetUsername(providerOptions.Username).
		SetPassword(providerOptions.Password).
		SetOrderMatters(false).
		SetMaxReconnectInterval(connectTimeout).
		SetOnConnectHandler(func(client mqtt.Client) {
			log.Infof("Successfully connected to gateway MQTT at: %s", provider.options.Gateway)
			providerOptions.OnStarted <- &BrokerStarted{Started: true}
			for _, handler := range getTopicHandlers(provider) {
				token := client.Subscribe(handler.topic, 1, handler.MessageHandler)
				ok := token.WaitTimeout(connectTimeout)
				if !ok {
					log.Errorf("error subscribing to topic '%s': %v\n", handler.topic, ErrTimeout)
				} else if token.Error() != nil {
					log.Errorf("error subscribing to topic '%s': %v\n", handler.topic, token.Error())
				}
			}
		}).
		// mqtt library should handle reconnecting for us, so we just report the error
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			providerOptions.OnStarted <- &BrokerStarted{Started: false}
			log.Errorf("lost connection to gateway (will try to reconnect): %s", err)
		}).
		SetTLSConfig(
			&tls.Config{
				InsecureSkipVerify: providerOptions.AllowSelfSignedCerts,
			}))

	provider = &MosquittoProvider{
		options: providerOptions,
	}

	return
}

// Publish uses the mqtt client in the mosquitto provider to publish a json payload
// to a given MQTT topic
func (provider *MosquittoProvider) Publish(topic string, payload []byte) (err error) {
	logrus.Debugf("Publishing to topic %s content: %s", topic, payload)
	if token := provider.options.MttqClient.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		err = token.Error()
	}
	return err
}

//Start makes the mosquitto broker connect and start listening to the existing topics
func (provider *MosquittoProvider) Start(onItem ProviderItemChannel, onError ErrorChannel) {
	metricError := metrics.GetOrRegisterGauge("Rfid-Provider.MosquittoProvider.ConnectGateway.Error", nil)
	metricSuccess := metrics.GetOrRegisterGauge("Rfid-Provider.MosquittoProvider.ConnectGateway.Success", nil)

	provider.onItem = onItem
	provider.onError = onError

	retry := 10
	var err error

	token := provider.options.MttqClient.Connect()

	for retry > 0 {
		retry--
		ok := token.WaitTimeout(connectTimeout)
		if !ok {
			err = ErrTimeout
		} else if token.Error() != nil {
			ok = false
			err = token.Error()
		}

		if ok {
			metricSuccess.Update(1)
			break
		}

		log.Errorf("error connecting to gateway MQTT at %s: %s", provider.options.Gateway, err)

		if retry == 0 {
			metricError.Update(1)
			onError <- ErrorData{Error: fmt.Errorf("provider unable to connect to gateway MQTT at: %s", provider.options.Gateway)}
			break
		}

		time.Sleep(5 * time.Second)
	}
}

//Stop makes the mosquitto broker stop listening to topics
func (provider *MosquittoProvider) Stop() {
	provider.options.MttqClient.Disconnect(0)
}

// messageHandler returns a mqtt.MessageHandler for the given message type
func (provider *MosquittoProvider) messageHandler(messageType string) mqtt.MessageHandler {
	return func(client mqtt.Client, message mqtt.Message) {
		metricMessageSuccess := metrics.GetOrRegisterGauge("Rfid-Provider.MosquittoProvider.MessageHandler.Success", nil)

		if provider.onItem == nil {
			log.Info("OnItem channel is null. Nowhere to send messages.")
			return
		}

		log.Debugf("Received MQTT message (%s): %s\n", messageType, string(message.Payload()))

		metricMessageSuccess.Update(1)
	}
}

type topicHandler struct {
	topic string
	mqtt.MessageHandler
}

func getTopicHandlers(p *MosquittoProvider) []topicHandler {
	return []topicHandler{
		{ProductItemTopic, p.messageHandler(ProductItemType)},
	}
}
