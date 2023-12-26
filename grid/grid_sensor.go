package grid

import (
	"fmt"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
	"github.com/foogod/go-powerwall"
)

type Sensor struct {
	*accessory.A
	sensor *service.ContactSensor
	client *powerwall.Client
}

func NewSensor(client *powerwall.Client) *Sensor {
	info := accessory.Info{Name: "Grid Power"}

	sensor := &Sensor{client: client}
	sensor.A = accessory.New(info, accessory.TypeSensor)
	sensor.sensor = service.NewContactSensor()
	sensor.AddS(sensor.sensor.S)

	sensor.sensor.ContactSensorState.SetValue(sensor.getSensorState())
	// sensor.sensor.ContactSensorState.OnValueRemoteGet(sensor.getSensorState)

	return sensor
}

func (s *Sensor) getSensorState() int {
	gridConnectionStatus, err := s.client.GetGridStatus()

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
