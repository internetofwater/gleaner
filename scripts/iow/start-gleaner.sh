#!/bin/sh




TS=`date +%Y-%m-%dT%H.%M.%S`
LOGDIR="$HOME/logs/$TS"
mkdir -p $LOGDIR || exit 1
cd $LOGDIR || exit 1

for src in `cat ~/conf/gleanerconfig.yaml | grep '\Wname:'|awk '{print $2}'`
do

OUTFILE="$LOGDIR/gleaner-$src.out"
ERRFILE="$LOGDIR/gleaner-$src.err"

echo "harvesting source '$src'..."
#strace -f -o $LOGDIR/strace-$src.out gleaner -cfg $HOME/conf/gleanerconfig.yaml -source $src -rude > $OUTFILE 2>$ERRFILE
gleaner -log debug  -cfg $HOME/conf/gleanerconfig.yaml -source $src -rude > $OUTFILE 2>$ERRFILE
done
echo "complete!"

