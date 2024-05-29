#!/bin/sh


docker run -d  \
	--rm \
	-p 9222:9222 \
	--name hc \
	chromedp/headless-shell

