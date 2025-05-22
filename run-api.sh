#!/bin/bash

pkill -f english-learning-api
source .env
nohup ./bin/english-learning-api >> english_learning_api.log 2>&1 &