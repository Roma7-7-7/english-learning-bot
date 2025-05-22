#!/bin/bash

pkill -f english-learning-bot
source .env
nohup ./bin/english-learning-bot >> english_learning_bot.log 2>&1 &