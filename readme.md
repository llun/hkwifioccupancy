# hkwifioccupancy

Wifi Occupancy Sensor homekit accessory for [HomeControl](https://github.com/brutella/hc)

## Sample Code for using with HC

```golang
package main

import (
  "github.com/brutella/hc"
  "github.com/brutella/hc/accessory"
  "github.com/llun/hkwifioccupancy"
)

func main() {
  wifi := wifioccupancy.NewWifi("/tmp/presence.wifi", NewSet("MAC ADDRESS1", "MAC ADDRESS2"))

  t, err := hc.NewIPTransport(hc.Config{
    Pin:  "32191123",
  }, wifi.Accessory)
  if err != nil {
    log.Fatal(err)
  }

  hc.OnTermination(func() {
    t.Stop()
  })

  t.Start()
}
```

# License

MIT
