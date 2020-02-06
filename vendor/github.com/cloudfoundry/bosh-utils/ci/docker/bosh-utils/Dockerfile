FROM golang:1

RUN \
  apt-get update \
  && apt-get install -y \
    lsof \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
