#!/usr/bin/env bash
boot datanode weed -max=100 -mserver=${MSERVER} -disk=hdd -dir=/data/hdd -metricsPort=9325
