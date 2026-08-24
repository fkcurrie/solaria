package main

// SunCondition represents the categorized state of solar harvest
type SunCondition string

const (
	ConditionFullSun         SunCondition = "FULL_SUN"
	ConditionPartialSun      SunCondition = "PARTIAL_SUN_OR_SHADE"
	ConditionDiffuseOvercast SunCondition = "DIFFUSE_OVERCAST"
	ConditionAbsorptionFloat SunCondition = "ABSORPTION_FLOAT_CLIPPED"
	ConditionNight           SunCondition = "NIGHT"
	ConditionVariable        SunCondition = "VARIABLE_SUN"
)

// ClassifySunCondition correlates telemetry with ambient irradiance
func ClassifySunCondition(telem *Telemetry, weather WeatherMetrics, panelRatedWatts float64) SunCondition {
	if telem == nil {
		return ConditionNight
	}

	if telem.PVPowerW <= 2 && (!weather.IsDay || telem.PVVoltageV < 5.0) {
		return ConditionNight
	}

	if telem.BatterySOCPct >= 99 && (telem.ChargingState == "Floating Charging" || telem.ChargingState == "Boost Charging") {
		return ConditionAbsorptionFloat
	}

	directRad := weather.DirectRadiationWM2
	diffuseRad := weather.DiffuseRadiationWM2
	cloudCover := weather.CloudCoverPct

	if panelRatedWatts <= 0 {
		panelRatedWatts = 400.0
	}
	harvestRatio := float64(telem.PVPowerW) / panelRatedWatts

	if cloudCover < 25 && directRad > 300 && harvestRatio > 0.65 {
		return ConditionFullSun
	}

	if cloudCover > 80 || (diffuseRad > directRad && directRad < 150) {
		return ConditionDiffuseOvercast
	}

	if (cloudCover >= 25 && cloudCover <= 80) || harvestRatio < 0.60 {
		return ConditionPartialSun
	}

	return ConditionVariable
}
