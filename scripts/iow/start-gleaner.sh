#!/bin/sh




CFGPATH="configs/iow/pids-geoconnex-dev-gleanerconfig.yaml"

for src in `cat $CFGPATH | grep '\Wname:'|awk '{print $2}'`
do


echo "harvesting source '$src'..."
./gleaner -log debug  -cfg $CFGPATH -source $src -rude
done
echo "complete!"

