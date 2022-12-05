package smartie

type ShellyStatus struct {
	ID      int     `json:"id"`
	Source  string  `json:"source"`
	Output  bool    `json:"output"`
	Apower  float64 `json:"apower"`
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Aenergy struct {
		Total    float64   `json:"total"`
		ByMinute []float64 `json:"by_minute"`
		MinuteTs int       `json:"minute_ts"`
	} `json:"aenergy"`
	Temperature struct {
		TC float64 `json:"tC"`
		TF float64 `json:"tF"`
	} `json:"temperature"`
}

type TasmotaStatus struct {
	Time string `json:"Time"`
	SML  struct {
		VerbrauchT1      float64 `json:"Verbrauch_T1"`
		VerbrauchT2      float64 `json:"Verbrauch_T2"`
		VerbrauchSumme   float64 `json:"Verbrauch_Summe"`
		EinspeisungSumme float64 `json:"Einspeisung_Summe"`
		WattL1           float64 `json:"Watt_L1"`
		WattL2           float64 `json:"Watt_L2"`
		WattL3           float64 `json:"Watt_L3"`
		WattSumme        float64 `json:"Watt_Summe"`
		VoltL1           float64 `json:"Volt_L1"`
		VoltL2           float64 `json:"Volt_L2"`
		VoltL3           float64 `json:"Volt_L3"`
	} `json:"SML"`
}
