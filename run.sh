#!/bin/bash

make build
if [ $? -ne 0 ]; then
	echo "build failed"
	exit 1
fi
pkill -f english-learning-bot
source .env
nohup ./bin/english-learning-bot >> english_learning_bot.log 2>&1 &