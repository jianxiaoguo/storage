#!/usr/bin/env bash
boot mainnode weed -mdir=/data -ip=$(hostname -I | tr -d ' ') -port=9333 -peers=$(hostname -I | tr -d ' '):9333 -metricsPort=9324
