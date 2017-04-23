package wifioccupancy

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/brutella/hc/log"
	"github.com/rjeczalik/notify"

	. "github.com/deckarep/golang-set"
)

type FilePresence struct {
	file      string
	addresses Set
}

func NewFilePresence(file string, addresses Set) *FilePresence {
	return &FilePresence{file, addresses}
}

func (p *FilePresence) Watch(monitor chan<- bool) error {
	if _, err := os.Stat(p.file); os.IsNotExist(err) {
		_, err = os.Create(p.file)
		if err != nil {
			return err
		}
	}

	eventCh := make(chan notify.EventInfo, 1)
	if err := notify.Watch(p.file, eventCh, notify.Write); err != nil {
		return err
	}

	monitor <- p.IsOccupied()
	go func() {
		defer notify.Stop(eventCh)
		for range eventCh {
			time.AfterFunc(time.Second, func() {
				monitor <- p.IsOccupied()
			})
		}
	}()
	return nil
}

func (p *FilePresence) IsOccupied() bool {
	data, err := ioutil.ReadFile(p.file)
	if err != nil {
		return false
	}

	addresses := strings.Split(string(data), "\n")
	currentSet := NewSet()
	for _, address := range addresses {
		currentSet.Add(address)
	}

	isOccupied := p.addresses.Intersect(currentSet).Cardinality() > 0
	log.Debug.Printf("Current addresses\n%v\nCurrent set\n%v\nIntersection\n%v\nend", p.addresses, currentSet, p.addresses.Intersect(currentSet))
	log.Debug.Printf("Is presence detected? %v", isOccupied)
	return isOccupied
}
