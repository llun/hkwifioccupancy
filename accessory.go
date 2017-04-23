package wifioccupancy

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"

	. "github.com/deckarep/golang-set"
)

type Presence interface {
	Watch(monitor chan<- bool) error
	IsOccupied() bool
}

type Sensor struct {
	*accessory.Accessory

	OccupancySensor *service.OccupancySensor

	presence  Presence
	addresses Set
}

func NewSensor(file string, addresses Set) *Sensor {
	info := accessory.Info{
		Name:         "WifiOccupancy",
		Manufacturer: "Compex",
		SerialNumber: "6A02",
		Model:        "WPQ864",
	}
	acc := Sensor{
		Accessory: accessory.New(info, accessory.TypeOther),
		addresses: addresses,
	}
	acc.OccupancySensor = acc.createOccupancySensorSevice()
	acc.AddService(acc.OccupancySensor.Service)

	if file != "" {
		acc.presence = NewFilePresence(file, addresses)
	} else {
		acc.presence = NewNetlinkPresence(addresses)
	}
	acc.watch()

	log.Info.Println("Wifi occupancy sensor is ready")
	return &acc
}

func (s *Sensor) AddAddress(address string) {
	s.addresses.Add(address)
}

func (s *Sensor) RemoveAddress(address string) {
	s.addresses.Remove(address)
}

func (s *Sensor) createOccupancySensorSevice() *service.OccupancySensor {
	sensor := service.NewOccupancySensor()
	detector := sensor.OccupancyDetected
	detector.SetValue(characteristic.OccupancyDetectedOccupancyNotDetected)
	return sensor
}

func (s *Sensor) watch() {
	isOccupiedCh := make(chan bool, 16)
	err := s.presence.Watch(isOccupiedCh)
	if err != nil {
		log.Info.Fatal(err)
	}

	sensor := s.OccupancySensor
	detector := sensor.OccupancyDetected
	go func() {
		for isOccupied := range isOccupiedCh {
			if isOccupied {
				detector.SetValue(characteristic.OccupancyDetectedOccupancyDetected)
			} else {
				detector.SetValue(characteristic.OccupancyDetectedOccupancyNotDetected)
			}
		}
	}()
}
