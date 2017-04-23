package wifioccupancy

import (
	"errors"
	"net"
	"os"

	"github.com/brutella/hc/log"
	"github.com/hkwi/nlgo"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"

	. "github.com/deckarep/golang-set"
)

type NetlinkPresence struct {
	addresses Set
}

func NewNetlinkPresence(addresses Set) *NetlinkPresence {
	return &NetlinkPresence{addresses}
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

		for _, address := range addresses {
			log.Info.Println(address.String())
		}

		go func() {
			defer connection.Close()

			for {
				messages, _, err := connection.Receive()
				if err != nil {
					log.Info.Fatalf("failed to receive messages: %v", err)
				}

				for _, message := range messages {
					header := message.Header
					switch header.Command {
					case nlgo.NL80211_CMD_DEL_STATION:
						log.Info.Println("Delete station")
						attrs, _ := p.getAttributes(message.Data)
						log.Info.Println(net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data).String())
					case nlgo.NL80211_CMD_NEW_STATION:
						log.Info.Println("New station")
						attrs, _ := p.getAttributes(message.Data)
						log.Info.Println(net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data).String())
					}
				}
			}
		}()
	}

	return nil
}

func (p *NetlinkPresence) IsOccupied() bool {
	return false
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

func (p *NetlinkPresence) getAllStations(connection *genetlink.Conn) ([]net.HardwareAddr, error) {
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

		var stations []net.HardwareAddr
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

				stations = append(stations, net.HardwareAddr(attrs[nlgo.NL80211_ATTR_MAC].Data))
			}
		}
		return stations, nil
	}
}
