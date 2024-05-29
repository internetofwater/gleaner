#!/bin/sh





for src in `cat configs/iow/pids-geoconnex-dev-gleanerconfig.yaml | grep '\Wname:'|awk '{print $2}'`
do


echo "harvesting source '$src'..."
./gleaner -log debug  -cfg config/gleanerconfig.yaml -source $src -rude
done
echo "complete!"

