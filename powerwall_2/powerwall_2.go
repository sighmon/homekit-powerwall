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
	outlet  *service.Outlet
	client  *powerwall.Client
}

func NewPowerwall2(client *powerwall.Client) *Powerwall2 {
	info := accessory.Info{
		Name:         "Powerwall 2",
		Model:        "3012170-05-E",
		Manufacturer: "Tesla",
		SerialNumber: "TG123000000ABC",
		Firmware:     "1.0.0",
	}

	powerwall := &Powerwall2{client: client}
	powerwall.A = accessory.New(info, accessory.TypeOutlet)
	powerwall.battery = service.NewBatteryService()
	powerwall.AddS(powerwall.battery.S)
	powerwall.outlet = service.NewOutlet()
	powerwall.AddS(powerwall.outlet.S)

	powerwall.battery.BatteryLevel.SetValue(powerwall.getChargePercentage())
	powerwall.battery.BatteryLevel.OnValueRemoteUpdate(powerwall.updateBatteryLevel)

	powerwall.battery.ChargingState.SetValue(powerwall.getChargingState())
	powerwall.battery.ChargingState.OnValueRemoteUpdate(powerwall.updateChargingState)

	powerwall.battery.StatusLowBattery.SetValue(powerwall.getLowBatteryStatus())
	powerwall.battery.StatusLowBattery.OnValueRemoteUpdate(powerwall.updateStatusLowBattery)

	// set outlet on if it's charging or exporting
	powerwall.outlet.On.SetValue(powerwall.getChargingOrExporting())
	powerwall.outlet.On.OnValueRemoteUpdate(powerwall.updateOutletOn)

	// set outlet in use if it's exporting
	powerwall.outlet.OutletInUse.SetValue(powerwall.getExporting())
	powerwall.outlet.OutletInUse.OnValueRemoteUpdate(powerwall.updateOutletInUse)

	return powerwall
}

func (pw *Powerwall2) updateBatteryLevel(v int) {
	currentCharge := pw.getChargePercentage()
	pw.battery.BatteryLevel.SetValue(currentCharge)
}

func (pw *Powerwall2) updateChargingState(v int) {
	currentChargeState := pw.getChargingState()
	pw.battery.ChargingState.SetValue(currentChargeState)
}

func (pw *Powerwall2) updateStatusLowBattery(v int) {
	currentLowBatteryState := pw.getLowBatteryStatus()
	pw.battery.StatusLowBattery.SetValue(currentLowBatteryState)
}

func (pw *Powerwall2) getChargePercentage() int {
	batteryStatus, err := pw.client.GetSOE()
	if err != nil {
		fmt.Printf("getChargePercentage error: %+v\n", err)

		return -1
	}
	rounded := math.RoundToEven(float64(batteryStatus.Percentage))

	return int(rounded)
}

func (pw *Powerwall2) getChargingState() int {
	chargingStatus, err := pw.client.GetMetersAggregates()
	if err != nil {
		fmt.Printf("getChargingState error: %+v\n", err)
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

func (pw *Powerwall2) updateOutletOn(v bool) {
	pw.outlet.On.SetValue(pw.getChargingOrExporting())
}

func (pw *Powerwall2) updateOutletInUse(v bool) {
	pw.outlet.OutletInUse.SetValue(pw.getExporting())
}

func (pw *Powerwall2) getChargingOrExporting() bool {
	chargingStatus, err := pw.client.GetMetersAggregates()
	if err != nil {
		fmt.Printf("getChargingOrExporting error: %+v\n", err)
		return false
	}

	charge := pw.battery.BatteryLevel.Value()

	if charge == 100 {
		// battery is fully charged
		return false
	} else if (*chargingStatus)["battery"].InstantPower < 0 {
		// battery is charging
		return true
	}

	// battery is discharging
	return true
}

func (pw *Powerwall2) getExporting() bool {
	chargingStatus, err := pw.client.GetMetersAggregates()
	if err != nil {
		fmt.Printf("getExporting error: %+v\n", err)
		return false
	}

	if (*chargingStatus)["battery"].InstantPower > 0 {
		// battery is discharging
		return true
	}

	return false
}
