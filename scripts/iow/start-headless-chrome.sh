#!/bin/sh

set -e # exit on error

docker run -d  \
	--rm \
	-p 9222:9222 \
	--name hc \
	chromedp/headless-shell

