package loganomalyprocessor

import (
	"time"

	"go.uber.org/zap"
)

type AnomalyFormatter struct {
	lastReportTime time.Time
	reportBuffer   []AnomalyStat
}

func newAnomalyFormatter() *AnomalyFormatter {
	return &AnomalyFormatter{
		lastReportTime: time.Now(),
		reportBuffer:   make([]AnomalyStat, 0),
	}
}

func (af *AnomalyFormatter) Add(stat AnomalyStat) {
	af.reportBuffer = append(af.reportBuffer, stat)
}

func (af *AnomalyFormatter) shouldReport() bool {
	return len(af.reportBuffer) >= 5 || time.Since(af.lastReportTime) >= time.Minute
}

func (af *AnomalyFormatter) LogReport(logger *zap.Logger) {
	if len(af.reportBuffer) == 0 {
		return
	}

	// Log each anomaly as a separate, structured log entry
	for _, stat := range af.reportBuffer {
		icon := "📈"
		if stat.anomalyType == "Drop" {
			icon = "📉"
		}

		logger.Info("Log anomaly detected",
			zap.String("anomaly_type", icon+" "+stat.anomalyType),
			zap.Float64("baseline_rate", stat.baselineRate),
			zap.Float64("current_rate", stat.currentRate),
			zap.Float64("deviation_percentage", stat.percentageDiff))
	}
}

func (af *AnomalyFormatter) Clear() {
	af.reportBuffer = af.reportBuffer[:0]
	af.lastReportTime = time.Now()
}
