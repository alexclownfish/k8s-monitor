#!/bin/bash
#node_exporter_status_scripts
status=`systemctl status node_exporter | grep "Active" | awk '{print $2}'`

if [ $status=="active" ]
then
  echo "node_exporter_status 0"
else
  echo "node_exporter_status 1"
fi
#alertgo_status_scripts

alertgostatus=`lsof -i:8088`

if [ "$?" = 0 ]
then
  echo "alertgo_status 0"
else
  echo "alertgo_status 1"
fi
