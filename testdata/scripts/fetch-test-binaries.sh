#!/bin/sh
curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.2/kubebuilder_2.3.2_linux_amd64.tar.gz | tar xvz --strip-components=1 -C testdata/
ls -l testdata/bin/
