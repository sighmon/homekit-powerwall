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
	Battery     *service.BatteryService
	Outlet      *service.Outlet
	Load        *service.LightSensor
	Solar       *service.LightSensor
	client      *powerwall.Client
	meters      *map[string]powerwall.MeterAggregatesData
	metersError error
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
	powerwall.Battery = service.NewBatteryService()
	powerwall.AddS(powerwall.Battery.S)
	powerwall.Outlet = service.NewOutlet()
	powerwall.AddS(powerwall.Outlet.S)
	powerwall.Load = service.NewLightSensor()
	powerwall.AddS(powerwall.Load.S)
	powerwall.Solar = service.NewLightSensor()
	powerwall.AddS(powerwall.Solar.S)

	powerwall.Battery.BatteryLevel.SetValue(powerwall.getChargePercentage())
	powerwall.Battery.BatteryLevel.OnValueRemoteUpdate(powerwall.updateBatteryLevel)

	powerwall.Battery.ChargingState.SetValue(powerwall.getChargingState())
	powerwall.Battery.ChargingState.OnValueRemoteUpdate(powerwall.updateChargingState)

	powerwall.Battery.StatusLowBattery.SetValue(powerwall.getLowBatteryStatus())
	powerwall.Battery.StatusLowBattery.OnValueRemoteUpdate(powerwall.updateStatusLowBattery)

	// set outlet on if it's charging or exporting
	powerwall.Outlet.On.SetValue(powerwall.getChargingOrExporting())
	powerwall.Outlet.On.OnValueRemoteUpdate(powerwall.updateOutletOn)
	powerwall.Outlet.OutletInUse.SetValue(powerwall.getChargingOrExporting())
	powerwall.Outlet.OutletInUse.OnValueRemoteUpdate(powerwall.updateOutletInUse)

	// add light sensor for the current load measurement
	powerwall.Load.CurrentAmbientLightLevel.SetMinValue(0)
	powerwall.Load.CurrentAmbientLightLevel.SetMaxValue(10000)
	powerwall.Load.CurrentAmbientLightLevel.SetValue(powerwall.getCurrentLoad())
	powerwall.Load.CurrentAmbientLightLevel.OnValueRemoteUpdate(powerwall.updateCurrentLoad)

	// add light sensor for the current solar measurement
	powerwall.Solar.CurrentAmbientLightLevel.SetMinValue(0)
	powerwall.Solar.CurrentAmbientLightLevel.SetMaxValue(10000)
	powerwall.Solar.CurrentAmbientLightLevel.SetValue(powerwall.getCurrentSolar())
	powerwall.Solar.CurrentAmbientLightLevel.OnValueRemoteUpdate(powerwall.updateCurrentSolar)

	return powerwall
}

func (pw *Powerwall2) UpdateAll() {
	pw.updateBatteryLevel(0)
	pw.updateChargingState(0)
	pw.updateStatusLowBattery(0)
	pw.updateOutletOn(true)
	pw.updateOutletInUse(true)
	pw.updateCurrentLoad(0)
	pw.updateCurrentSolar(0)
}

func (pw *Powerwall2) updateBatteryLevel(v int) {
	currentCharge := pw.getChargePercentage()
	pw.Battery.BatteryLevel.SetValue(currentCharge)
}

func (pw *Powerwall2) updateChargingState(v int) {
	currentChargeState := pw.getChargingState()
	pw.Battery.ChargingState.SetValue(currentChargeState)
}

func (pw *Powerwall2) updateStatusLowBattery(v int) {
	currentLowBatteryState := pw.getLowBatteryStatus()
	pw.Battery.StatusLowBattery.SetValue(currentLowBatteryState)
}

func (pw *Powerwall2) updateOutletOn(v bool) {
	pw.Outlet.On.SetValue(pw.getChargingOrExporting())
}

func (pw *Powerwall2) updateOutletInUse(v bool) {
	pw.Outlet.OutletInUse.SetValue(pw.getChargingOrExporting())
}

func (pw *Powerwall2) updateCurrentLoad(v float64) {
	pw.Load.CurrentAmbientLightLevel.SetValue(pw.getCurrentLoad())
}

func (pw *Powerwall2) updateCurrentSolar(v float64) {
	pw.Solar.CurrentAmbientLightLevel.SetValue(pw.getCurrentSolar())
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
	pw.meters, pw.metersError = pw.client.GetMetersAggregates()
	if pw.metersError != nil {
		fmt.Printf("getChargingState error: %+v\n", pw.metersError)
		return -1
	}

	charge := pw.Battery.BatteryLevel.Value()

	if charge == 100 {
		// battery is fully charged
		return characteristic.ChargingStateNotCharging
	} else if (*pw.meters)["battery"].InstantPower < 0 {
		// battery is charging
		return characteristic.ChargingStateCharging
	}

	// battery is discharging
	return characteristic.ChargingStateNotCharging
}

func (pw *Powerwall2) getLowBatteryStatus() int {
	charge := pw.Battery.BatteryLevel.Value()

	if charge <= 5 {
		return characteristic.StatusLowBatteryBatteryLevelLow
	}

	return characteristic.StatusLowBatteryBatteryLevelNormal
}

func (pw *Powerwall2) getChargingOrExporting() bool {
	if pw.metersError != nil {
		fmt.Printf("getChargingOrExporting error: %+v\n", pw.metersError)
		return false
	}

	charge := pw.Battery.BatteryLevel.Value()
	batteryPower := (*pw.meters)["battery"].InstantPower

	if batteryPower > 0 && charge < 100 {
		// battery is discharging
		return true
	} else if batteryPower < 0 {
		// battery is charging
		return true
	} else if charge == 100 {
		// battery is fully charged
		return false
	}

	return true
}

func (pw *Powerwall2) getCurrentLoad() float64 {
	if pw.metersError != nil {
		fmt.Printf("getCurrentLoad error: %+v\n", pw.metersError)
		return -1
	}

	return float64((*pw.meters)["load"].InstantPower)
}

func (pw *Powerwall2) getCurrentSolar() float64 {
	if pw.metersError != nil {
		fmt.Printf("getCurrentSolar error: %+v\n", pw.metersError)
		return -1
	}

	return float64((*pw.meters)["solar"].InstantPower)
}
