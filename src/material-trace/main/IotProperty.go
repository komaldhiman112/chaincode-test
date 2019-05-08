package main

/*
Defines a structure for properties expected from IOT event hub
*/
type IotProperty struct {
	// IotId         string  `json:"iotId"`
	TrackingCode  string  `json:"trackingCode"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Humidity      float64 `json:"humidity"`
	Accelerometer float64 `json:"accelerometer"`
	Temperature   float64 `json:"temperature"`
	Timestamp     int64   `json:"timestamp"`
}
