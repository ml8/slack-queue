#!/bin/bash

gcloud compute instances create-with-container ${SQ_VM} \
  --container-image gcr.io/${SQ_GCP_PROJECT}/slack-queue:${SQ_VERSION_TAG} \
  --machine-type f1-micro \
  --tags app \
  --address ${SQ_GCP_IP} \
  --disk name=${SQ_DATA_DISK},mode=rw \
  --container-mount-disk mount-path="/disks/data-disk",name=${SQ_DATA_DISK},mode=rw \
  --container-arg="-oauth=${SQ_OAUTH}" \
  --container-arg="-ssecret=${SQ_SIGNING_SECRET}" \
  --container-arg="-csecret=${SQ_CLIENT_SECRET}" \
  --container-arg="-listCommand=list" \
  --container-arg="-putCommand=tutorme" \
  --container-arg="-takeCommand=tutor" \
  --container-arg="-authChannel=faculty" \
  --container-arg="-stateFilename=/disks/data-disk/qstate/state" \
  --container-arg="-logtostderr"
  
