package coe

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/app/config"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultSiteId string = "UnknownSite"
)

type responseData struct {
	Data DeviceInformation `json:"data"`
}

type DeviceInformation struct {
	DeviceId     string `json:"deviceId"`
	SiteId       string `json:"deviceName"`
	PotalUrl     string `json:"potalUrl"`
	PotalAuthUrl string `json:"potalAuthUrl"`
	DomainName   string `json:"domainName"`
	HostName     string `json:"hostName"`
}

var DeviceInfo = DeviceInformation{SiteId: DefaultSiteId}

// get the preferred outbound ip addr of this system
func getOutboundIP() net.IP {
	// this ip address can be anything non-local, only a virtual connection is made, no data is exchanged
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Warn("Failed to close connection")
		}
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func loadFromCache() ([]byte, error) {
	jsonFile, err := os.Open(config.AppConfig.DeviceInfoCacheFile)
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully Opened users.json")

	defer func() {
		if err := jsonFile.Close(); err != nil {
			log.Warning("unable to close cache file")
		}
	}()

	return ioutil.ReadAll(jsonFile)
}

func loadFromApi() ([]byte, error) {
	// determine what ip to call the service on
	host := getOutboundIP()
	log.Debugf("host ip: %s", host)
	deviceApi := fmt.Sprintf("https://%s/b.service/api/device", host)
	log.Debugf("device api: %s", deviceApi)

	resp, err := http.Get(deviceApi)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET returned non-200 status: %d.", resp.StatusCode)
	}
	if resp.Body == nil {
		return nil, errors.Errorf("No response body from server")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method": "InitializeSiteId",
			}).Warning("Failed to close response.")
		}
	}()

	jsonResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return jsonResponse, nil
}

func InitializeDeviceInfo() error {
	jsonResponse, err := loadFromApi()
	if err != nil {
		log.Info("Unable to load device info from api server. Attempting to load from cache")

		if jsonResponse, err = loadFromCache(); err != nil {
			log.WithError(err).Debug("Unable to load device info from cache file")
			return err
		}
		log.Debugf("Successfully loaded device info from cache file: %s", config.AppConfig.DeviceInfoCacheFile)
	}

	log.Debugf("JSON device info: %s", jsonResponse)

	respData := responseData{}
	err = json.Unmarshal(jsonResponse, &respData)
	if err != nil {
		return errors.Wrap(err, "Unable to unmarshal response from server")
	}

	DeviceInfo = respData.Data

	// cache to file
	if err = ioutil.WriteFile(config.AppConfig.DeviceInfoCacheFile, jsonResponse, 0644); err != nil {
		return errors.Wrap(err, "Unable to cache device information to file")
	}

	return nil
}
