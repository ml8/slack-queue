#!/bin/bash

docker build -t slack-queue:${SQ_VERSION_TAG} -f Dockerfile .
docker tag slack-queue:${SQ_VERSION_TAG} gcr.io/${SQ_GCP_PROJECT}/slack-queue:${SQ_VERSION_TAG}
docker push gcr.io/${SQ_GCP_PROJECT}/slack-queue:${SQ_VERSION_TAG}
