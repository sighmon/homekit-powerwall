package grid

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

var httpClient *http.Client

func init() {
	// ignore bad SSL certificates for the powerwall :(
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient = &http.Client{
		Transport: transCfg,
		Timeout:   time.Second * 2,
	}
}

type Sensor struct {
	*accessory.A

	sensor *service.ContactSensor

	ip net.IP
}

func NewSensor(ip net.IP) *Sensor {
	info := accessory.Info{Name: "Grid Power"}

	sensor := &Sensor{ip: ip}
	sensor.A = accessory.New(info, accessory.TypeSensor)
	sensor.sensor = service.NewContactSensor()
	sensor.AddS(sensor.sensor.S)

	sensor.sensor.ContactSensorState.SetValue(sensor.getSensorState())
	// sensor.sensor.ContactSensorState.OnValueRemoteGet(sensor.getSensorState)

	return sensor
}

func (s *Sensor) makeRequest(uri string, ret interface{}) error {
	url := fmt.Sprintf("https://%s%s", s.ip.String(), uri)

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(ret)
	if err != nil {
		return err
	}

	return nil
}

type apiGridConnectionStatus struct {
	GridStatus string `json:"grid_status"`
}

func (s *Sensor) getSensorState() int {
	gridConnectionStatus := &apiGridConnectionStatus{}

	err := s.makeRequest("/api/system_status/grid_status", gridConnectionStatus)
	if err != nil {
		fmt.Printf("getSensorState error: %+v\n", err)

		return -1
	}

	switch gridConnectionStatus.GridStatus {
	case "SystemIslandedActive": // grid is down
		return characteristic.ContactSensorStateContactNotDetected
	case "SystemGridConnected": // grid is up
		fallthrough
	case "SystemTransitionToGrid": // grid is restored but not yet in sync
		fallthrough
	default:
		return characteristic.ContactSensorStateContactDetected
	}
}
