package model

import "time"

type OrderPaidEvent struct {
	UUID      string
	OrderUUID string
	UserUUID  string
}

type ShipAssembledEvent struct {
	UUID         string
	OrderUUID    string
	UserUUID     string
	BuildTimeSec int64
	AssembledAt  time.Time
}
