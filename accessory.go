package wifioccupancy

import (
  "io/ioutil"
  "os"
  "strings"
  "time"

  "github.com/brutella/hc/accessory"
  "github.com/brutella/hc/characteristic"
  "github.com/brutella/hc/log"
  "github.com/brutella/hc/service"
  "github.com/rjeczalik/notify"

  . "github.com/deckarep/golang-set"
)

type Sensor struct {
  *accessory.Accessory

  OccupancySensor *service.OccupancySensor

  file      string
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
    file:      file,
    addresses: addresses,
  }
  acc.OccupancySensor = acc.createOccupancySensorSevice()
  acc.AddService(acc.OccupancySensor.Service)
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

func (s *Sensor) watch() {
  if _, err := os.Stat(s.file); os.IsNotExist(err) {
    _, err = os.Create(s.file)
    if err != nil {
      log.Debug.Fatal(err)
    }
  }

  eventCh := make(chan notify.EventInfo, 1)
  if err := notify.Watch(s.file, eventCh, notify.Write); err != nil {
    log.Debug.Fatal(err)
  }

  sensor := s.OccupancySensor
  detector := sensor.OccupancyDetected
  if s.isOccupied() {
    detector.SetValue(characteristic.OccupancyDetectedOccupancyDetected)
  } else {
    detector.SetValue(characteristic.OccupancyDetectedOccupancyNotDetected)
  }

  go func() {
    defer notify.Stop(eventCh)
    for range eventCh {
      time.AfterFunc(time.Second, func() {
        if s.isOccupied() {
          detector.SetValue(characteristic.OccupancyDetectedOccupancyDetected)
        } else {
          detector.SetValue(characteristic.OccupancyDetectedOccupancyNotDetected)
        }
      })
    }
  }()
}

func (s *Sensor) createOccupancySensorSevice() *service.OccupancySensor {
  sensor := service.NewOccupancySensor()
  detector := sensor.OccupancyDetected
  detector.SetValue(characteristic.OccupancyDetectedOccupancyNotDetected)
  return sensor
}

func (s *Sensor) isOccupied() bool {
  data, err := ioutil.ReadFile(s.file)
  if err != nil {
    log.Info.Fatal(err)
  }

  addresses := strings.Split(string(data), "\n")
  currentSet := NewSet()
  for _, address := range addresses {
    currentSet.Add(address)
  }

  isOccupied := s.addresses.Intersect(currentSet).Cardinality() > 0
  log.Debug.Printf("Current addresses\n%v\nCurrent set\n%v\nIntersection\n%v\nend", s.addresses, currentSet, s.addresses.Intersect(currentSet))
  log.Debug.Printf("Is presence detected? %v", isOccupied)
  return isOccupied
}
