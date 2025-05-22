#!/bin/bash

make build
if [ $? -ne 0 ]; then
	echo "build failed"
	exit 1
fi
pkill -f english-learning-api
source .env
nohup ./bin/english-learning-api >> english_learning_api.log 2>&1 &