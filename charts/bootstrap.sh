#!/bin/bash

kubectl create -f helm-bootstrap.yaml
helm init --service-account helm
