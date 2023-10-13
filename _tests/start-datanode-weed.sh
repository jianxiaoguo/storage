#!/usr/bin/env bash
boot datanode weed -max=100 -mserver="${MSERVER}" -disk=hdd -dir=/data -metricsPort=9325
