ARG CODENAME
FROM registry.drycc.cc/drycc/base:${CODENAME}

ENV DRYCC_UID=1001 \
  DRYCC_GID=1001 \
  DRYCC_HOME_DIR=/home/drycc \
  JQ_VERSION="1.7.1" \
  TIKV_VERSION="7.5.0" \
  SEAWEEDFS_VERSION="3.61" \
  SEAWEEDFS_DATA_DIR=/data \
  SEAWEEDFS_CONF_DIR=/etc/seaweedfs

RUN groupadd drycc --gid ${DRYCC_GID} \
  && useradd drycc -u ${DRYCC_UID} -g ${DRYCC_GID} -s /bin/bash -m -d ${DRYCC_HOME_DIR} \
  && install-stack jq $JQ_VERSION \
  && install-stack tikv $TIKV_VERSION \
  && install-stack seaweedfs $SEAWEEDFS_VERSION \
  && mkdir -p ${SEAWEEDFS_DATA_DIR} ${SEAWEEDFS_CONF_DIR} \
  && chown -hR ${DRYCC_UID}:${DRYCC_GID} ${SEAWEEDFS_CONF_DIR} \
  && chown -hR ${DRYCC_UID}:${DRYCC_GID} ${SEAWEEDFS_DATA_DIR}

USER ${DRYCC_UID}

COPY --chown=${DRYCC_UID}:${DRYCC_GID} rootfs/bin /bin

ENTRYPOINT ["init-stack", "/bin/boot"]
