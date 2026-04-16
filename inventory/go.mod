// TODO: Поменяй имя модуля github.com/student на своё и обнови все импорты
module github.com/vixart/rocket-factory/inventory

go 1.26.0

require (
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.42.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

replace github.com/vixart/rocket-factory/shared => ./../shared
