#!/bin/sh

MC="$1"
DIR="$2"
DESIRED_COUNT="$3"

if [ "$#" -ne 3 ]
then
   echo "Usage: $0 MC_CMD_PATH DIR DESIRED_COUNT" && exit 1
fi


ACTUAL_COUNT=`$MC ls $DIR| wc -l`

if [ "$DESIRED_COUNT" -ne "$ACTUAL_COUNT" ]
then
     echo "$DIR: wanted $DESIRED_COUNT directory entries, found $ACTUAL_COUNT"
fi

