#!/bin/bash

docker build -t slack-queue -f Dockerfile .
docker tag slack-queue:latest gcr.io/${SQ_GCP_PROJECT}/slack-queue
docker push gcr.io/${SQ_GCP_PROJECT}/slack-queue

gcloud compute instances update-container slack-queue \
  --container-image gcr.io/${SQ_GCP_PROJECT}/slack-queue:latest \
  --container-arg="-oauth=${SQ_OAUTH}" \
  --container-arg="-ssecret=${SQ_SIGNING_SECRET}" \
  --container-arg="-csecret=${SQ_CLIENT_SECRET}" \
  --container-arg="-listCommand=list" \
  --container-arg="-putCommand=tutorme" \
  --container-arg="-takeCommand=tutor" \
  --container-arg="-authChannel=faculty" \
  --container-arg="-stateFilename=/disks/data-disk/qstate/state" \
  --container-arg="-logtostderr"
  
