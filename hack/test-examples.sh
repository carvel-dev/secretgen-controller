#!/bin/bash

set -e -x -u

time kapp deploy -y -a certs -f examples/certs.yml
time kapp deploy -y -a certs -f examples/certs-rotation
time kapp delete -y -a certs

time kapp deploy -y -a passwords -f examples/passwords.yml
time kapp delete -y -a passwords

time kapp deploy -y -a rsa-key -f examples/rsa-key.yml
time kapp delete -y -a rsa-key

time kapp deploy -y -a secret-export-image-pull-secret -f examples/secret-export-image-pull-secret.yml
time kapp delete -y -a secret-export-image-pull-secret

time kapp deploy -y -a secret-export -f examples/secret-export.yml
time kapp delete -y -a secret-export

time kapp deploy -y -a ssh-key -f examples/ssh-key.yml
time kapp delete -y -a ssh-key

time kapp deploy -y -a secret-template -f examples/secret-template.yml
time kapp delete -y -a secret-template

time kapp deploy -y -a secret-template-non-secret-inputs -f examples/secret-template-non-secret-inputs.yml
time kapp delete -y -a secret-template-non-secret-inputs

echo SUCCESS
