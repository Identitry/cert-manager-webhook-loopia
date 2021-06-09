# Cert-Manager ACME webhook for Loopia (cert-manager-webhook-loopia)

`cert-manager-webhook-loopia` is an ACME webhook for [Cert-Manager](https://cert-manager.io/) that allows for [Cert-Manager] to use `DNS-01` challenge against the [Loopia](https://loopia.com) DNS.

[![test](https://github.com/Identitry/cert-manager-webhook-loopia/actions/workflows/test.yml/badge.svg)](https://github.com/Identitry/cert-manager-webhook-loopia/actions/workflows/test.yml)

[![release](https://github.com/Identitry/cert-manager-webhook-loopia/actions/workflows/release.yml/badge.svg)](https://github.com/Identitry/cert-manager-webhook-loopia/actions/workflows/release.yml)

## Table of Contents

1. [Overview](#1-overview)

   1.1. [Building](#11-building)

   1.2. [Docker Image](#12-docker-image)

   1.3. [Compatibility](#13-compatibility)
2. [Installation](#2-installation)

   2.2. [Install Cert-Manager](#22-install-cert-manager)

   2.3. [Install/Uninstall Loopia Webhook](#23-install/uninstall-loopia-webhook)

3. [Using the Loopia Webhook](#3-using-the-loopia-webhook)

   3.1. [Loopia API credential Secret](#31-loopia-api-credential-secret)

   3.2. [Cert-Manager Issuer configuration](#32-cert-manager-issuer-configuration)

   3.3. [Cert-Manager Certificate configuration](#33-cert-manager-certificate-configuration)

   3.4. [Troubleshooting](#34-troubleshooting)
4. [Conformance Testing](#4-conformance-testing)

## 1. Overview

[Cert-Manager](https://cert-manager.io) is a [Kubernetes](https://kubernetes.io/) certificate management controller, it allows for issuing certifaces from a variety of sources. `cert-manager-webhook-loopia` that acts as an extension to cert-manager is targeted the use of certificates issued through the [ACME-protocol](https://en.wikipedia.org/wiki/Automated_Certificate_Management_Environment) and especially the DNS01 challenge targeting the [Loopia](https://loopia.com) hosting company DNS. Issued certificates are stored in Kubernetes as [secrets](https://kubernetes.io/docs/concepts/configuration/secret) for use within Kubernetes.

The main issuer of public certificates using the ACME-protocol is [Let´s Encrypt](https://letsencrypt.org) that issues free public TLS-certificates. What´s special about Let´s Encrypt is that the lifetime of the certificates they issue are short and issuance is automated using the ACME-protocol, that means you need a way to automatically request new certificates and renew certificates when the old are about expire.

The [ACME DNS-01 challenge](https://letsencrypt.org/docs/challenge-types/#dns-01-challenge) is one of two diiferent challenges (the other is [HTTP-01](https://letsencrypt.org/docs/challenge-types/#http-01-challenge)) that you as a user of Let´s Encrypt certificates could use to prove you´re the owner of the domain the certificate is to be issued for. ACME DNS-01 challenge has an advantage over HTTP-01 in that it allows for issuance of wildcard certificates. ACME DNS-01 challenge requires you to be able to automatically add a DNS TXT record to your public DNS zone as a proof of ownership of the domain.

The role of `cert-manager-webhook-loopia` is to act as a DNS-provider and create a DNS TXT-record in the '\_acme-challenge' sub domain of the domain a certificate should be issued for, for example: '\_acme-challenge.example.com'. The value the TXT-record should contain is supplied by the ACME issuer. When the DNS01 challenge is complete, `cert-manager-webhook-loopia` is responsible for cleaning up the TXT-records created. Currently `cert-manager-webhook-loopia` can´t delete the '\_acme-challenge' sub domain due to lack of functionality in the [Loopia-Go client](https://github.com/jonlil/loopia-go) used but TXT records are removed.

[Loopia](https://loopia.com) is a major hosting company based in Sweden but has subsidaries in Norway and Serbia but also offers services to companies and individuals in the rest of the world.

[Loopia API](https://www.loopia.com/api) that is used by `cert-manager-webhook-loopia` is an API based on XMLRPC that allows for reading and editing of your DNS domain(s) hosted at Loopia. This API becomes very handy when we need to request a lot of certificates automatically and also renew these when they expire. [Loopia-Go client](https://github.com/jonlil/loopia-go) is the client library used for communicating with Loopia API.

---
**NOTE** You need to register for special Loopia API user credentials in  [Loopia CustomerZone](https://customerzone.loopia.com/), this is also required for testing.

---

![Loopia API Logo](https://static.loopia.se/loopiaweb/images/logos/loopia-api-logo.png "Loopia API Logo")

### 1.1. Building

Build the container image `cert-manager-webhook-loopia:latest`

```shell
make build
```

### 1.2. Docker Image

An image is hosted on Docker Hub:
[identitry/cert-manager-webhook-loopia](https://hub.docker.com/repository/docker/identitry/cert-manager-webhook-loopia)

### 1.3. Compatibility

This webhook has been tested with [cert-manager] v1.2.0 and Kubernetes v1.20.x on `amd64`.

## 2. Installation

### 2.1. Prereqs

Before starting the installation of `cert-manager-webhook-loopia` the prerequisite is that you have a working Kubernetes cluster, either in the cloud or on bare metal.
You could of course use [Cert-Manager] and `cert-manager-webhook-loopia` on [Minikube](https://minikube.sigs.k8s.io/docs/), [Microk8s](https://microk8s.io/), [K3s](https://k3s.io/) or [Docker Desktop with Kubernetes enabled](https://www.docker.com/products/docker-desktop).
The installation also require that you have registered for Loopia API credentials in the [Loopia CustomerZone](https://customerzone.loopia.com), these special credentials are required for `cert-manager-webhook-loopia` to work.

### 2.2. Install Cert-Manager

The easiest way to install Cert-Manager is using Helm. For this Helm v3 needs to be installed already.
This is how to install Cert-Manager using Helm, if you wish to install using manifests or using other options you can use this [instruction](https://cert-manager.io/docs/installation/kubernetes).

Add the Jetstack Helm Repository:

```shell
helm repo add jetstack https://charts.jetstack.io
```

Update Helm chart repository cache:

```shell
helm repo update
```

Install Cert-Manager (with CRD´s):

```shell
helm install cert-manager jetstack/cert-manager --namespace cert-manager --version v1.2.0 --create-namespace --set installCRDs=true
```

Verify Cert-Manager installation by getting the cert-manager running pods:

```shell
kubectl get pods --namespace cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-85f9bbcd97-666mx              1/1     Running   0          2m
cert-manager-cainjector-74459fcc56-r6dc8   1/1     Running   0          2m
cert-manager-webhook-57d97ccc67-jngx8      1/1     Running   0          2m
```

Note that it might take a minute or two before all pods are running.

### 2.3. Install/Uninstall Loopia Webhook

The `cert-manager-webhook-loopia` can be installed in multiple ways but the easiest is using helm:

```shell
helm repo add identitry https://identitry.github.io/cert-manager-webhook-loopia
helm repo update
helm install cert-manager-webhook-loopia identitry/cert-manager-webhook-loopia --namespace cert-manager
```

This will install a helm chart with the pre built image available in Docker Hub as identitry/cert-manager-webhook-loopia.

If you wish to uninstall `cert-manager-webhook-loopia` simply run this command:

```shell
helm uninstall cert-manager-webhook-loopia --namespace cert-manager
```

## 3. Using the Loopia Webhook

Ok, now you have probably installed `cert-manager-webhook-loopia`, it´s time to configure it for getting a certificate from Let´s Encrypt.

### 3.1. Loopia API credential Secret

In order to logon to the [Loopia API](https://www.loopia.com/api) you first need a set of credentials, as a customer with Loopia you can request these in the [Loopia Customer Zone](https://customerzone.loopia.com), the usual credentials we normally use to logon with Loopia wont work.

---

**Note:**
Your Loopia API account requires these permissions:

- addZoneRecord
- getZoneRecords
- removeZoneRecord
- removeSubdomain

---

When we have the Loopia API credentials (username and password) we need to store these credentials safely within Kubernetes and Kubernetes has a special API object type, [Secret](https://kubernetes.io/docs/concepts/configuration/secret) that can be used for this.

The Secret needs to be created in the "cert-manager" namespace, otherwise permissions needs to be given for cert-manager to use the Secret.
This is the secret configuration we need to apply to Kubernetes, you can find this file in the configuration/ folder. Replace the username and password with your Loopia API credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: loopia-credentials
  namespace: cert-manager
stringData:
  username: "LOOPIA API USERNAME"
  password: "LOOPIA API PASSWORD"
```

Then deploy the Secret to Kubernetes using this command:

```shell
kubectl apply -f configuration/loopia-credentials.yaml
```

You can also deploy the secret using this command, replace the username and password with your Loopia API credentials.

```shell
kubectl create secret generic loopia-credentials --namespace cert-manager --from-literal=username='LOOPIA API USERNAME' --from-literal=password='LOOPIA API PASSWORD'
```

To remove the Secret, run this command:

```shell
kubectl delete secret loopia-credentials --namespace cert-manager
```

### 3.2 Cert-Manager Issuer configuration

Issuers and ClusterIssuers, are Kubernetes resources that represent certificate authorities (CAs) that are able to generate signed certificates. An Issuer is limited to a single namespace whereas a ClusterIssuer can issue certificates for the whole cluster.
The example yaml below is for creating a ClusterIssuer but you can just change "ClusterIssuer" to "Issuer" if you like to restrict the certificate to a single namespace.

You also need to replace the email adress to your real email adress, Let´s Encrypt needs this to identify you as a subscriber and holder of the private key. This email adress will also recieve warnings of expiring certs and notifications about changes to [Let´s Encrypts privacy policy](https://letsencrypt.org/privacy).

The ClusterIssuer example below is targeted [Let´s Encrypts staging environment](https://letsencrypt.org/docs/staging-environment), this will allow you to get things right before issuing trusted certificates and reduce the chance of your running up against rate limits.
When you have successfully tested your configuration you can remove the staging ClusterIssuer and replace it with a production one pointing to the Let´s Encrypt production environment, changing the name and the name of the Secret where the issued certificate should end up.

The example below is also available as configuration/le-staging-clusterissuer.yaml.

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # The ACME server URL for testing.
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    # The ACME server URL for production.
    # server: https://acme-v02.api.letsencrypt.org/directory

    # You must replace this email address with your own.
    # Let's Encrypt will use this to contact you about expiring
    # certificates, and issues related to your account.
    email: hostmaster@example.com

    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging

    solvers:
      - dns01:
          webhook:
            groupName: acme.webhook.loopia.com
            solverName: loopia
            config:
              usernameSecretKeyRef:
                name: loopia-credentials
                key: username
              passwordSecretKeyRef:
                name: loopia-credentials
                key: password
```

To deploy the Cluster Issuer configuration file after you have edit it you can run this command:

```shell
kubectl apply -f configuration/le-staging-clusterissuer.yaml
```

Afterwards, check the status of the Cluster Issuer.

```shell
kubectl describe clusterissuer letsencrypt-staging
```

To delete the Cluster Issuer, run this command:

```shell
kubectl delete clusterissuer letsencrypt-staging
```

### 3.3. Cert-Manager Certificate configuration

Time to wrap this up, the final Kubernetes resource we need is a Certificate. The [Certificate resource](https://cert-manager.io/docs/usage/certificate) represents a human readable definition of a certificate request that is to be honored by an issuer which is to be kept up-to-date.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: staging-cert-example-com
spec:
  commonName: example.com # REPLACE THIS WITH YOUR DOMAIN
  dnsNames:
  - example.com # REPLACE THIS WITH YOUR DOMAIN
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
  secretName: example-com-tls
```

To check the status of the certificate you can run this command:

```shell
kubectl describe certificate staging-cert-example-com
```

If you wish to delete the Certificate, run this command:

```shell
kubectl delete certificate staging-cert-example-com
```

### 3.4. Troubleshooting

Cert-Manager has a great page that describes how to do [troubleshooting](https://cert-manager.io/docs/faq/troubleshooting).

## 4. Conformance Testing

The testing of a cert-manager weebhook is a bit special and not a typical unit or integration test, instead there´s a test-fixure supplied that build up a complete Kubernetes control plane where testing is performed. This not only requires you to download a set of test binaries but also prepare some files for testing.

- **testdata/scripts/fetch-test-binaries.sh:**\
  Script for downloading test binaries, this script is limited to Linux/Amd64 but other OS/architecture versions are available.

- **testdata/loopia/config.json:**\
  This is a config file that basically informs the test fixture how to find the Kubernetes secret and keys that contains the Loopia API username and password.

- **testdata/loopia/loopia-credentials.yaml:**\
  A Kubernetes secret configuration that will be applied to the Kubernetes control plane during test. Real Loopia API credentials is required since the tests connects to Loopia creating a cert-manager-dns01-tests sub domain with a TXT-record.

- **testdata/bin:**\
  Folder location for the test-binaries.

`cert-manager-webhook-loopia` has been tested for conformance, not only simple create/delete TXT-record but also in Strict/Extended mode where multiple simultaneus TXT-records are tested.

[ACME DNS-01 challenge]: https://letsencrypt.org/docs/challenge-types/#dns-01-challenge
[ACME documentation]: https://cert-manager.io/docs/configuration/acme
[Certificate]: https://cert-manager.io/docs/usage/certificate
[Cert-Manager]: https://cert-manager.io
[Let´s Encrypt]: https://letsencrypt.org
[Loopia]: https://loopia.com/
[Loopia Customer Zone]: https://www.loopia.com/login
[Loopia API]: https://doc.livedns.loopia.com
[Helm]: https://helm.sh
[image tags]: https://hub.docker.com/repository/docker/identitry/cert-manager-webhook-loopia
[Kubernetes]: https://kubernetes.io
