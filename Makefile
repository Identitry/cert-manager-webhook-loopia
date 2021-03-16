OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

IMAGE_NAME := "cert-manager-webhook-loopia"
IMAGE_TAG := "latest"

KUBEBUILDER_VERSION=2.3.2

test: testdata/bin
	go test -v .

testdata/bin:
	curl -fsSL https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$(KUBEBUILDER_VERSION)/kubebuilder_$(KUBEBUILDER_VERSION)_$(OS)_$(ARCH).tar.gz | tar xvz --strip-components=1 -C testdata/

clean: clean-test

clean-kubebuilder:
	rm -rf testdata/bin

build:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
	helm template \
	    --name cert-manager-webhook-loopia \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
        deploy/cert-manager-webhook-loopia > "$(OUT)/rendered-manifest.yaml"