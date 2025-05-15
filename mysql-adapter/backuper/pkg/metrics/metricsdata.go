package metrics

import "time"

type MetricsData struct {
	Name      string
	Success   bool
	Timestamp int64
}

func NewMetricsData(backupName string, success bool) MetricsData {
	return MetricsData{
		Name:      backupName,
		Success:   success,
		Timestamp: time.Now().Unix(),
	}
}
