package wifioccupancy

import (
	"errors"
	"net"
	"os"
	"time"

	"github.com/brutella/hc/log"
	"github.com/hkwi/nlgo"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"

	. "github.com/deckarep/golang-set"
)

type NetlinkPresence struct {
	watchedSet Set
	currentSet Set
}

func NewNetlinkPresence(addresses Set) *NetlinkPresence {
	return &NetlinkPresence{
		watchedSet: addresses,
	}
}

func (p *NetlinkPresence) Watch(monitor chan<- bool) error {
	connection, err := genetlink.Dial(nil)
	if err != nil {
		return err
	}

	if family, err := connection.Family.Get(nlgo.NL80211_GENL_NAME); err != nil {
		if os.IsNotExist(err) {
			log.Info.Printf("%q family not available", nlgo.NL80211_GENL_NAME)
		}

		log.Info.Printf("failed to query for family: %v", err)
		return err
	} else {
		log.Info.Printf("%s: %+v", nlgo.NL80211_GENL_NAME, family)
		groups := p.getMulticastGroups(family.Groups)
		group, ok := groups["mlme"]
		if !ok {
			return errors.New("mlme group is not found")
		}

		err := connection.JoinGroup(group.ID)
		if err != nil {
			log.Info.Println("Cannot join mlme")
			return err
		}

		addresses, err := p.getAllStations(connection)
		if err != nil {
			return err
		}

		macAddresses := make([]interface{}, len(addresses))
		for idx, value := range addresses {
			macAddresses[idx] = value
		}

		p.currentSet = NewSet(macAddresses...)
		monitor <- p.IsOccupied()

		go p.ReceivingNetlinkEvent(connection, monitor)
		go p.PollingStation(monitor)
	}

	return nil
}

func (p *NetlinkPresence) ReceivingNetlinkEvent(connection *genetlink.Conn, monitor chan<- bool) {
	for {
		messages, _, err := connection.Receive()
		if err != nil {
			log.Debug.Printf("failed to receive messages: %v", err)
		}

		for _, message := range messages {
			header := message.Header
			switch header.Command {
			case nlgo.NL80211_CMD_DEL_STATION:
				attrs, _ := p.getAttributes(message.Data)
				station := net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data).String()
				p.currentSet.Remove(station)
				log.Info.Printf("%v is disconnected", station)
				monitor <- p.IsOccupied()
			case nlgo.NL80211_CMD_NEW_STATION:
				attrs, _ := p.getAttributes(message.Data)
				station := net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data).String()
				p.currentSet.Add(station)
				log.Info.Printf("%v is connected", station)
				monitor <- p.IsOccupied()
			}

		}
	}
}

func (p *NetlinkPresence) PollingStation(monitor chan<- bool) {
	connection, err := genetlink.Dial(nil)
	if err != nil {
		log.Debug.Printf("Cannot polling because of error: %v", err)
		return
	}

	// For device deauthenticated without disassociated
	tickerCh := time.Tick(5 * time.Second)
	for range tickerCh {
		addresses, err := p.getAllStations(connection)
		if err != nil {
			continue
		}

		macAddresses := make([]interface{}, len(addresses))
		for idx, value := range addresses {
			macAddresses[idx] = value
		}

		p.currentSet = NewSet(macAddresses...)
		monitor <- p.IsOccupied()
	}
}

func (p *NetlinkPresence) IsOccupied() bool {
	log.Debug.Println("Current: ", p.currentSet)
	log.Debug.Println("Watched: ", p.watchedSet)
	// TODO: Add option to check all/atleast one.
	// Current rule is all
	existing := p.currentSet.Intersect(p.watchedSet)
	log.Debug.Printf("Existing: %v, %v, %v\n", existing, existing.Cardinality(), p.watchedSet.Cardinality())
	isOccupied := existing.Cardinality() == p.watchedSet.Cardinality()
	log.Debug.Println("isOccupied: ", isOccupied)

	return isOccupied
}

func (p *NetlinkPresence) getMulticastGroups(groups []genetlink.MulticastGroup) map[string]genetlink.MulticastGroup {
	groupMap := make(map[string]genetlink.MulticastGroup)
	for _, group := range groups {
		groupMap[group.Name] = group
	}
	return groupMap
}

func (p *NetlinkPresence) getAttributes(data []byte) (map[uint16]netlink.Attribute, error) {
	if attrs, err := netlink.UnmarshalAttributes(data); err != nil {
		return nil, err
	} else {
		attributes := make(map[uint16]netlink.Attribute)
		for _, attr := range attrs {
			attributes[attr.Type] = attr
		}
		return attributes, nil
	}
}

func (p *NetlinkPresence) getAllStations(connection *genetlink.Conn) ([]string, error) {
	if family, err := connection.Family.Get(nlgo.NL80211_GENL_NAME); err != nil {
		if os.IsNotExist(err) {
			log.Info.Printf("%q family not available", nlgo.NL80211_GENL_NAME)
		}

		log.Info.Printf("failed to query for family: %v", err)
		return nil, err
	} else {
		req := genetlink.Message{
			Header: genetlink.Header{
				Command: nlgo.NL80211_CMD_GET_INTERFACE,
				Version: family.Version,
			},
		}

		flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
		msgs, err := connection.Execute(req, family.ID, flags)
		if err != nil {
			return nil, err
		}

		var stations []string
		for _, msg := range msgs {
			req = genetlink.Message{
				Header: genetlink.Header{
					Command: nlgo.NL80211_CMD_GET_STATION,
					Version: family.Version,
				},
				Data: msg.Data,
			}

			msgs, err = connection.Execute(req, family.ID, flags)
			if err != nil {
				return nil, err
			}

			for _, msg := range msgs {
				attrs, err := p.getAttributes(msg.Data)
				if err != nil {
					log.Info.Printf("failed to unmarshal attributes: %v", err)
					return nil, err
				}

				stations = append(stations, net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data).String())
			}
		}
		return stations, nil
	}
}
