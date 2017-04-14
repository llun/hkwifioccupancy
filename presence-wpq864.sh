#!/bin/sh
while true
do
  all=`date`
  for interface in `iwconfig | grep 802.11 | cut -f 1 -s -d" "`
  do
    maclist=`wlanconfig $interface list | tail +2 | cut -f 1 -s -d" "`
    for mac in $maclist
    do
      all="$mac\n$all"
    done
  done
  echo -e $all | tee /tmp/presence.wifi
  sleep 5
done
