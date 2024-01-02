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
	Sensor *service.ContactSensor
	client *powerwall.Client
}

func NewSensor(client *powerwall.Client) *Sensor {
	info := accessory.Info{Name: "Grid Power"}

	sensor := &Sensor{client: client}
	sensor.A = accessory.New(info, accessory.TypeSensor)
	sensor.Sensor = service.NewContactSensor()
	sensor.AddS(sensor.Sensor.S)

	sensor.Sensor.ContactSensorState.SetValue(sensor.getSensorState())
	sensor.Sensor.ContactSensorState.OnValueRemoteUpdate(sensor.UpdateSensorState)

	return sensor
}

func (s *Sensor) UpdateSensorState(v int) {
	currentSensorState := s.getSensorState()
	s.Sensor.ContactSensorState.SetValue(currentSensorState)
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
