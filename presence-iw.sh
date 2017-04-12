#!/bin/sh
while true
do
  all=`date`
  for interface in `iw dev | grep Interface | cut -f 2 -s -d" "`
  do
    # for each interface, get mac addresses of connected stations/clients
    maclist=`iw dev $interface station dump | grep Station | cut -f 2 -s -d" "`

    # for each mac address in that list...
    for mac in $maclist
    do
      all="$all\n$mac"
    done
  done
  echo -e $all | tee /tmp/presence.wifi
sleep 5
done
