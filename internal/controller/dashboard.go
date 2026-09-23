package controller

import "encoding/json"

type GrafanaDashboard struct {
	UID           string         `json:"uid"`
	Title         string         `json:"title"`
	Tags          []string       `json:"tags"`
	Time          GrafanaTime    `json:"time"`
	Timezone      string         `json:"timezone"`
	SchemaVersion int            `json:"schemaVersion"`
	Version       int            `json:"version"`
	Panels        []GrafanaPanel `json:"panels"`
	Refresh       string         `json:"refresh"`
}

type GrafanaTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type GrafanaPanel struct {
	ID          int                    `json:"id"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	GridPos     GrafanaGridPos         `json:"gridPos"`
	Targets     []GrafanaTarget        `json:"targets"`
	Options     map[string]interface{} `json:"options,omitempty"`
	FieldConfig map[string]interface{} `json:"fieldConfig,omitempty"`
}

type GrafanaGridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

type GrafanaTarget struct {
	Expr  string `json:"expr"`
	RefID string `json:"refId"`
}

func buildGrafanaDashboard(
	deploymentName string,
) (string, error) {

	dashboard := GrafanaDashboard{
		UID:   deploymentName + "-observability",
		Title: deploymentName + " Dashboard",
		Tags: []string{
			"kubernetes",
			"observability",
		},
		Time: GrafanaTime{
			From: "now-1h",
			To:   "now",
		},
		Timezone:      "browser",
		SchemaVersion: 39,
		Version:       1,
		Refresh:       "10s",

		Panels: []GrafanaPanel{
			{
				ID:    1,
				Title: "CPU Usage",
				Type:  "timeseries",

				GridPos: GrafanaGridPos{
					H: 8,
					W: 12,
					X: 0,
					Y: 0,
				},

				Targets: []GrafanaTarget{
					{
						Expr:  `sum(rate(container_cpu_usage_seconds_total{pod=~"` + deploymentName + `-.*"}[5m]))`,
						RefID: "A",
					},
				},
			},

			{
				ID:    2,
				Title: "Memory Usage",
				Type:  "timeseries",

				GridPos: GrafanaGridPos{
					H: 8,
					W: 12,
					X: 12,
					Y: 0,
				},

				Targets: []GrafanaTarget{
					{
						Expr:  `sum(container_memory_working_set_bytes{pod=~"` + deploymentName + `-.*"})`,
						RefID: "A",
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
