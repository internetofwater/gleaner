#!/bin/sh

# exit on error
set -e

# set relative path of gleaner config from root of this repository
CFGPATH="configs/iow/pids-geoconnex-dev-gleanerconfig.yaml"

for src in `cat $CFGPATH | grep '\Wname:'|awk '{print $2}'`
do


echo "harvesting source '$src'..."
./gleaner -log info  -cfg $CFGPATH -source $src -rude
done
echo "complete!"

