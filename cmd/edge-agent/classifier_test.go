package main

import "testing"

func TestClassifySunCondition(t *testing.T) {
	tests := []struct {
		name     string
		telem    *Telemetry
		weather  WeatherMetrics
		ratedW   float64
		expected SunCondition
	}{
		{
			name: "Night condition",
			telem: &Telemetry{
				PVPowerW:   0,
				PVVoltageV: 0.5,
			},
			weather: WeatherMetrics{
				IsDay: false,
			},
			ratedW:   400.0,
			expected: ConditionNight,
		},
		{
			name: "Full sun high generation",
			telem: &Telemetry{
				PVPowerW:      320,
				PVVoltageV:    36.5,
				BatterySOCPct: 75,
				ChargingState: "MPPT Charging",
			},
			weather: WeatherMetrics{
				IsDay:               true,
				CloudCoverPct:       10,
				DirectRadiationWM2:  650,
				DiffuseRadiationWM2: 80,
			},
			ratedW:   400.0,
			expected: ConditionFullSun,
		},
		{
			name: "Absorption float clipped (SOC 99%)",
			telem: &Telemetry{
				PVPowerW:        30,
				PVVoltageV:      37.0,
				BatterySOCPct:   99,
				BatteryVoltageV: 13.6,
				ChargingState:   "Floating Charging",
			},
			weather: WeatherMetrics{
				IsDay:               true,
				CloudCoverPct:       5,
				DirectRadiationWM2:  700,
				DiffuseRadiationWM2: 70,
			},
			ratedW:   400.0,
			expected: ConditionAbsorptionFloat,
		},
		{
			name: "Diffuse overcast condition",
			telem: &Telemetry{
				PVPowerW:      35,
				PVVoltageV:    33.0,
				BatterySOCPct: 60,
			},
			weather: WeatherMetrics{
				IsDay:               true,
				CloudCoverPct:       95,
				DirectRadiationWM2:  50,
				DiffuseRadiationWM2: 120,
			},
			ratedW:   400.0,
			expected: ConditionDiffuseOvercast,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySunCondition(tt.telem, tt.weather, tt.ratedW)
			if got != tt.expected {
				t.Errorf("ClassifySunCondition() = %v, want %v", got, tt.expected)
			}
		})
	}
}
