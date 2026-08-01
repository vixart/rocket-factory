package metrics

// Config — конфигурация для инициализации логгера.
type Config struct {
	// ServiceName — имя сервиса, прикрепляется к каждому логу через resource (service.name)
	ServiceName string
	// Environment — окружение развёртывания (например, "production", "staging", "development")
	Environment string
	// InstanceID - имя хоста
	InstanceID string
	// CollectorEndpoint — адрес OTLP-коллектора (например, "localhost:4317")
	// Если не задан, используется defaultOTLPEndpoint
	CollectorEndpoint string
}
