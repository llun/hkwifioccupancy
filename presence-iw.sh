#!/bin/sh
while true
do
  currentTime=`date`
  all=""
  for interface in `iw dev | grep Interface | cut -f 2 -s -d" "`
  do
    # for each interface, get mac addresses of connected stations/clients
    maclist=`iw dev $interface station dump | grep Station | cut -f 2 -s -d" "`

    # for each mac address in that list...
    for mac in $maclist
    do
      if [ -z $all ]; then
        all=$mac
      else
        all="$all\n$mac"
      fi
    done
  done
  if [ ! -z $all ]; then
    echo -e "$currentTime\n$all" | tee /tmp/presence.wifi
  fi
sleep 5
done
