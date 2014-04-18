#!/bin/sh

set -e
set -x

DIR=releases
PACKAGE=sensord
SHA=$(git rev-parse HEAD)
BUILD=${DIR}/${PACKAGE}.${SHA}
TARGET=s3://canary-releases

if [ ! -d ${DIR} ]; then
  mkdir ${DIR}
fi

godep go install
godep go build

mv ${PACKAGE} ${BUILD}

s3cmd put ${BUILD} ${TARGET}
