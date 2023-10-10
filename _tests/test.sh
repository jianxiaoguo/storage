#!/usr/bin/env bash

BASE_DIR=$(dirname "$(readlink -f "$BASH_SOURCE")")
DRYCC_STORAGE_ACCESSKEY=f4c4281665bc11ee8e0400163e04a9cd
DRYCC_STORAGE_SECRETKEY=f4c4281665bc11ee8e0400163e04a9cd

function start-test-mainnode() {
  podman run -d --name test-logger-mainnode-tipd \
    --entrypoint init-stack \
    -v ${BASE_DIR}/bin:/usr/local/bin \
    -e DRYCC_STORAGE_JWT=5f0254bc65b511ee927c00163e04a9cd \
    registry.drycc.cc/drycc/storage:canary start-mainnode-tipd.sh

  podman run -d --name test-logger-mainnode-weed \
    --entrypoint init-stack \
    -v ${BASE_DIR}/bin:/usr/local/bin \
    -e DRYCC_STORAGE_JWT=5f0254bc65b511ee927c00163e04a9cd \
    registry.drycc.cc/drycc/storage:canary start-mainnode-weed.sh
}

function stop-test-mainnode() {
  podman kill test-logger-mainnode-tipd
  podman kill test-logger-mainnode-weed
  podman rm test-logger-mainnode-tipd
  podman rm test-logger-mainnode-weed
}

function start-test-metanode() {
  TIPD_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-mainnode-tipd)
  podman run -d --name test-logger-metanode-tikv \
    --entrypoint init-stack \
    -v ${BASE_DIR}/bin:/usr/local/bin \
    -e DRYCC_STORAGE_TIPD_ENDPOINTS=http://${TIPD_IP}:2379 \
    -e DRYCC_STORAGE_JWT=5f0254bc65b511ee927c00163e04a9cd \
    registry.drycc.cc/drycc/storage:canary start-metanode-tikv.sh

  WEED_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-mainnode-weed)
  podman run -d  --name test-logger-metanode-weed \
    --entrypoint init-stack \
    -v ${BASE_DIR}/bin:/usr/local/bin \
    -e MASTER=${WEED_IP}:9333 \
    -e DRYCC_STORAGE_JWT=5f0254bc65b511ee927c00163e04a9cd \
    -e DRYCC_STORAGE_ACCESSKEY=${DRYCC_STORAGE_ACCESSKEY} \
    -e DRYCC_STORAGE_SECRETKEY=${DRYCC_STORAGE_SECRETKEY} \
    -e DRYCC_STORAGE_TIPD_ENDPOINTS=http://${TIPD_IP}:2379 \
    registry.drycc.cc/drycc/storage:canary start-metanode-weed.sh
}

function stop-test-metanode() {
  podman kill test-logger-metanode-tikv
  podman kill test-logger-metanode-weed
  podman rm test-logger-metanode-tikv
  podman rm test-logger-metanode-weed
}

function start-test-datanode() {
  WEED_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-mainnode-weed)
  podman run -d --name test-logger-datanode-weed \
    --entrypoint init-stack \
    -v ${BASE_DIR}/bin:/usr/local/bin \
    -e MSERVER=${WEED_IP}:9333 \
    -e DRYCC_STORAGE_JWT=5f0254bc65b511ee927c00163e04a9cd \
    registry.drycc.cc/drycc/storage:canary start-datanode-weed.sh
}

function stop-test-datanode() {
  podman kill test-logger-datanode-weed
  podman rm test-logger-datanode-weed
}

function clean_before_exit {
  # delay before exiting, so stdout/stderr flushes through the logging system
  stop-test-mainnode
  stop-test-metanode
  stop-test-datanode
}
trap clean_before_exit EXIT

function main {
  start-test-mainnode
  start-test-metanode
  start-test-datanode
  S3_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-metanode-weed)
  S3_CLIENT=$(mktemp -u)
  S3_ENDPOINT=http://${S3_IP}:8333
  # wait for port
  echo -e "\\033[32m---> Waitting for ${S3_IP}:8333\\033[0m"
  WAIT_FOR_PORT_DIR=$(mktemp -d)
  curl -sSL https://github.com/bitnami/wait-for-port/releases/download/v1.0.7/wait-for-port-linux-$(dpkg --print-architecture).tar.gz | tar xvz -C ${WAIT_FOR_PORT_DIR}
  chmod +x ${WAIT_FOR_PORT_DIR}/wait-for-port-linux-$(dpkg --print-architecture)
  ${WAIT_FOR_PORT_DIR}/wait-for-port-linux-$(dpkg --print-architecture) --host=${S3_IP} 8333
  rm -rf ${WAIT_FOR_PORT_DIR}
  echo -e "\\033[32m---> S3 service ${S3_IP}:8333 ready...\\033[0m"
  # mc client
  echo -e "\\033[32m---> Downloading mc client...\\033[0m"
  curl -sfL https://dl.min.io/client/mc/release/linux-$(dpkg --print-architecture)/mc -o ${S3_CLIENT}
  chmod +x ${S3_CLIENT}
  ${S3_CLIENT} --config-dir /tmp/.mc config host add storage ${S3_ENDPOINT} ${DRYCC_STORAGE_ACCESSKEY} ${DRYCC_STORAGE_SECRETKEY} --lookup path --api s3v4
  ${S3_CLIENT} --config-dir /tmp/.mc cp ${BASE_DIR}/README.md storage/test
  exit_code=$?
  rm -rf /tmp/.mc ${S3_CLIENT}
  exit $exit_code
}

main
