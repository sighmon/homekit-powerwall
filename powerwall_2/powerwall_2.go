package powerwall_2

import (
	"fmt"
	"math"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
	"github.com/foogod/go-powerwall"
)

type Powerwall2 struct {
	*accessory.A

	battery *service.BatteryService
	client  *powerwall.Client
}

func NewPowerwall2(client *powerwall.Client) *Powerwall2 {
	// TODO: get powerwall info from the from the /api/powerwalls endpoint
	info := accessory.Info{
		Name:         "Powerwall 2",
		Model:        "2012170-00-A",
		Manufacturer: "Tesla",
		SerialNumber: "TG118252000S5W/TG118252000S65",
		Firmware:     "1.0.0",
	}

	powerwall := &Powerwall2{client: client}
	powerwall.A = accessory.New(info, accessory.TypeOther)
	powerwall.battery = service.NewBatteryService()
	powerwall.AddS(powerwall.battery.S)

	powerwall.battery.BatteryLevel.SetValue(powerwall.getChargePercentage())
	// powerwall.battery.BatteryLevel.OnValueRemoteUpdate(powerwall.getChargePercentage)

	powerwall.battery.ChargingState.SetValue(powerwall.getChargingState())
	// powerwall.battery.ChargingState.OnValueRemoteUpdate(powerwall.getChargingState)

	powerwall.battery.StatusLowBattery.SetValue(powerwall.getLowBatteryStatus())
	// powerwall.battery.StatusLowBattery.OnValueRemoteUpdate(powerwall.getLowBatteryStatus)

	return powerwall
}

func (pw *Powerwall2) getChargePercentage() int {
	batteryStatus, err := pw.client.GetSOE()
	if err != nil {
		fmt.Printf("updateChargePercentage error: %+v\n", err)

		return -1
	}
	rounded := math.RoundToEven(float64(batteryStatus.Percentage))

	return int(rounded)
}

func (pw *Powerwall2) getChargingState() int {
	chargingStatus, err := pw.client.GetMetersAggregates()
	if err != nil {
		fmt.Printf("updateChargingState error: %+v\n", err)
		return -1
	}

	charge := pw.battery.BatteryLevel.Value()

	if charge == 100 {
		// battery is fully charged
		return characteristic.ChargingStateNotChargeable
	} else if (*chargingStatus)["battery"].InstantPower < 0 {
		// battery is charging
		return characteristic.ChargingStateCharging
	}

	// battery is discharging
	return characteristic.ChargingStateNotCharging
}

func (pw *Powerwall2) getLowBatteryStatus() int {
	charge := pw.battery.BatteryLevel.Value()

	if charge <= 5 {
		return characteristic.StatusLowBatteryBatteryLevelLow
	}

	return characteristic.StatusLowBatteryBatteryLevelNormal
}
