# Cert-Manager ACME webhook for Loopia (cert-manager-webhook-loopia)

`cert-manager-webhook-loopia` is an ACME webhook for [Cert-Manager](https://cert-manager.io/) that allows for [Cert-Manager] to use `DNS-01` challenge against the [Loopia](https://loopia.com) DNS.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)

   2.1. [Prereqs](#prereqs)

   2.2. [Secret Loopia API credential](#creds)

   2.2. [Install Loopia Webhook](#webhookinstall)


<a name="overview"></a>
## 1. Overview

[Cert-Manager](https://cert-manager.io) is a [Kubernetes](https://kubernetes.io/) certificate management controller, it allows for issuing certifaces from a variety of sources. `cert-manager-webhook-loopia` that acts as an extension to cert-manager is targeted the use of certificates issued through the [ACME-protocol](https://en.wikipedia.org/wiki/Automated_Certificate_Management_Environment) and especially the DNS01 challenge targeting the [Loopia](https://loopia.com) hosting company DNS. Issued certificates are stored in Kubernetes as [secrets](https://kubernetes.io/docs/concepts/configuration/secret) for use within Kubernetes.

The main issuer of public certificates using the ACME-protocol is [Let´s Encrypt](https://letsencrypt.org) that issues free public TLS-certificates. What´s special about Let´s Encrypt is that the lifetime of the certificates they issue are short and issuance is automated using the ACME-protocol, that means you need a way to automatically request new certificates and renew certificates when the old are about expire.

The [ACME DNS-01 challenge](https://letsencrypt.org/docs/challenge-types/#dns-01-challenge) is one of two diiferent challenges (the other is [HTTP-01](https://letsencrypt.org/docs/challenge-types/#http-01-challenge)) that you as a user of Let´s Encrypt certificates could use to prove you´re the owner of the domain the certificate is to be issued for. ACME DNS-01 challenge has an advantage over HTTP-01 in that it allows for issuance of wildcard certificates. ACME DNS-01 challenge requires you to be able to automatically add a DNS TXT record to your public DNS zone as a proof of ownership of the domain.

[Loopia](https://loopia.com) is a major hosting company based in Sweden but has subsidaries in Norway and Serbia but also offers services to companies and individuals in the rest of the world.

[Loopia API](https://www.loopia.com/api) that is used by `cert-manager-webhook-loopia` is an API based on XMLRPC that allows for reading and editing of your DNS domain(s) hosted at Loopia. This API becomes very handy when we need to request a lot of certificates automatically and also renew these when they expire.

![Loopia API Logo](https://static.loopia.se/loopiaweb/images/logos/loopia-api-logo.png "Loopia API Logo")

<a name="installation"></a>
## 2. Installation

<a name="prereqs"></a>
### 2.1. Prereqs

Before starting the installation of `cert-manager-webhook-loopia` the prerequisite is that you have a working Kubernetes cluster, either in the cloud or on bare metal and that you have deployed Cert-Manager already on your cluster.
You could of course use [Cert-Manager] and `cert-manager-webhook-loopia` on [Minikube](https://minikube.sigs.k8s.io/docs/), [Microk8s](https://microk8s.io/), [K3s](https://k3s.io/) or [Docker Desktop with Kubernetes enabled](https://www.docker.com/products/docker-desktop).
The installation also require that you have registered for Loopia API credentials in the [Loopia CustomerZone], these special credentials are required for `cert-manager-webhook-loopia` to work.

<a name="prereqs"></a>
### 2.2. Secret Loopia API credential

In order to logon to the [Loopia API](https://www.loopia.com/api) you first need a set of credentials, as a customer with Loopia you can request these in the [Loopia Customer Zone], the usual credentials we normally use to logon with Loopia wont work.
When we have the Loopia API credentials (username and password) we need to store the credentials safely within Kubernetes and Kubernetes has a special API object type, [Secret](https://kubernetes.io/docs/concepts/configuration/secret) that can be used for this.

This is the secret configuration we need to apply to Kubernetes:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: loopia-api-credentials
stringData:
  username: <your Loopia API username goes here>
  password: <your Loopia API password goes here>
```

<a name="webhookinstall"></a>
### 2.3. Install Loopia Webhook
The `cert-manager-webhook-loopia` can be installed in multiple ways, these are the options:



## Building
Build the container image `cert-manager-webhook-loopia:latest`:

    make build


## Image
Ready made images are hosted on Docker Hub ([image tags]). Use at your own risk:

    bwolf/cert-manager-webhook-loopia


### Release History
Refer to the [ChangeLog](ChangeLog.md) file.


## Compatibility
This webhook has been tested with [cert-manager] v0.13.1 and Kubernetes v0.17.x on `amd64`. In theory it should work on other hardware platforms as well but no steps have been taken to verify this. Please drop me a note if you had success.


## Testing with Minikube
1. Build this webhook in Minikube:

        minikube start --memory=4G --more-options
        eval $(minikube docker-env)
        make build
        docker images | grep webhook

2. Install [cert-manager] with [Helm]:

        kubectl create namespace cert-manager
        kubectl apply --validate=false -f https://raw.githubusercontent.com/jetstack/cert-manager/v0.13.1/deploy/manifests/00-crds.yaml

        helm repo add jetstack https://charts.jetstack.io
        helm install cert-manager --namespace cert-manager \
            --set 'extraArgs={--dns01-recursive-nameservers=8.8.8.8:53\,1.1.1.1:53}' \
            jetstack/cert-manager

        kubectl get pods --namespace cert-manager --watch

    **Note**: refer to Name servers in the official [documentation][setting-nameservers-for-dns01-self-check] according the `extraArgs`.

    **Note**: ensure that the custom CRDS of cert-manager match the major version of the cert-manager release by comparing the URL of the CRDS with the helm info of the charts app version:

            helm search repo jetstack

    Example output:

            NAME                    CHART VERSION   APP VERSION     DESCRIPTION
            jetstack/cert-manager   v0.13.1         v0.13.1         A Helm chart for cert-manager

    Check the state and ensure that all pods are running fine (watch out for any issues regarding the `cert-manager-webhook-` pod  and its volume mounts):

            kubectl describe pods -n cert-manager | less


3. Create the secret to keep the Loopia API key in the default namespace, where later on the Issuer and the Certificate are created:

        kubectl create secret generic loopia-credentials \
            --from-literal=api-token='<LOOPIA-API-KEY>'

    **Note**: See [RBAC Authorization]:

    > A Role can only be used to grant access to resources within a single namespace.

    *As far as I understand cert-manager, the `Secret` must reside in the same namespace as the `Issuer` and `Certificate` resource.*

4. Grant permission for the service-account to access the secret holding the Loopia API key:

        kubectl apply -f rbac.yaml

5. Deploy this locally built webhook (add `--dry-run` to try it and `--debug` to inspect the rendered manifests; Set `logLevel` to 6 for verbose logs):

        helm install cert-manager-webhook-loopia \
            --namespace cert-manager \
            --set image.repository=cert-manager-webhook-loopia \
            --set image.tag=latest \
            --set logLevel=2 \
            ./deploy/cert-manager-webhook-loopia

    To deploy using the image from Docker Hub (for example using the `v0.1.1` tag):

        helm install cert-manager-webhook-loopia \
            --namespace cert-manager \
            --set image.tag=v0.1.1 \
            --set logLevel=2 \
            ./deploy/cert-manager-webhook-loopia

    Check the logs

            kubectl get pods -n cert-manager --watch
            kubectl logs -n cert-manager cert-manager-webhook-loopia-XYZ

6. Create a staging issuer (email addresses with the suffix `example.com` are forbidden):

        cat << EOF | sed "s/invalid@example.com/$email/" | kubectl apply -f -
        apiVersion: cert-manager.io/v1alpha2
        kind: Issuer
        metadata:
          name: letsencrypt-staging
          namespace: default
        spec:
          acme:
            # The ACME server URL
            server: https://acme-staging-v02.api.letsencrypt.org/directory
            # Email address used for ACME registration
            email: invalid@example.com
            # Name of a secret used to store the ACME account private key
            privateKeySecretRef:
              name: letsencrypt-staging
            solvers:
            - dns01:
                webhook:
                  groupName: acme.bwolf.me
                  solverName: loopia
                  config:
                    apiKeySecretRef:
                      key: api-token
                      name: loopia-credentials
        EOF

    Check status of the Issuer:

        kubectl describe issuer letsencrypt-staging

    *Note*: The production Issuer is [similar][ACME documentation].

7. Issue a [Certificate] for your `$DOMAIN`:

        cat << EOF | sed "s/example-com/$DOMAIN/" | kubectl apply -f -
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: example-com
        spec:
          dnsNames:
          - example-com
          issuerRef:
            name: letsencrypt-staging
          secretName: example-com-tls
        EOF

    Check the status of the Certificate:

        kubectl describe certificate $DOMAIN

    Display the details like the common name and subject alternative names:

        kubectl get secret $DOMAIN-tls -o yaml

8. Issue a wildcard Certificate for your `$DOMAIN`:

        cat << EOF | sed "s/example-com/$DOMAIN/" | kubectl apply -f -
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: wildcard-example-com
        spec:
          dnsNames:
          - '*.example-com'
          issuerRef:
            name: letsencrypt-staging
          secretName: wildcard-example-com-tls
        EOF

    Check the status of the Certificate:

        kubectl describe certificate $DOMAIN

    Display the details like the common name and subject alternative names:

        kubectl get secret wildcard-$DOMAIN-tls -o yaml

99. Uninstall this webhook:

        helm uninstall cert-manager-webhook-loopia --namespace cert-manager
        kubectl delete -f rbac.yaml
        kubectl delete loopia-credentials

100. Uninstalling cert-manager:
This is out of scope here. Refer to the official [documentation][cert-manager-uninstall].


## Development
**Note**: If some tool (IDE or build process) fails resolving a dependency, it may be the cause that a indirect dependency uses `bzr` for versioning. In such a case it may help to put the `bzr` binary into `$PATH` or `$GOPATH/bin`.


## Release process
- Code changes result in a new image version and Git tag
- Helm chart changes result in a new chart version
- All other changes are pushed to master
- All versions are to be documented in [ChangeLog](ChangeLog.md)


## Conformance test
Please note that the test is not a typical unit or integration test. Instead it invokes the web hook in a Kubernetes-like environment which asks the web hook to really call the DNS provider (.i.e. Loopia). It attempts to create an `TXT` entry like `cert-manager-dns01-tests.example.com`, verifies the presence of the entry via Google DNS. Finally it removes the entry by calling the cleanup method of web hook.

**Note**: Replace the string `darwin` in the URL below with an OS matching your system (e.g. `linux`).

As said above, the conformance test is run against the real Loopia API. Therefore you *must* have a Loopia account, a domain and an API key.

``` shell
cp testdata/loopia/api-key.yaml.sample testdata/loopia/api-key.yaml
echo -n $YOUR_LOOPIA_API_KEY | base64 | pbcopy # or xclip
$EDITOR testdata/loopia/api-key.yaml
./scripts/fetch-test-binaries.sh
TEST_ZONE_NAME=example.com. go test -v .
```


[ACME DNS-01 challenge]: https://letsencrypt.org/docs/challenge-types/#dns-01-challenge
[ACME documentation]: https://cert-manager.io/docs/configuration/acme
[Certificate]: https://cert-manager.io/docs/usage/certificate
[Cert-Manager]: https://cert-manager.io
[Let´s Encrypt]: https://letsencrypt.org
[Loopia]: https://loopia.com/
[Loopia Customer Zone]: https://www.loopia.com/login
[Loopia API]: https://doc.livedns.loopia.com
[Helm]: https://helm.sh
[image tags]: https://hub.docker.com/r/bwolf/cert-manager-webhook-loopia
[Kubernetes]: https://kubernetes.io/
[RBAC Authorization]: https://kubernetes.io/docs/reference/access-authn-authz/rbac
[setting-nameservers-for-dns01-self-check]: https://cert-manager.io/docs/configuration/acme/dns01/#setting-nameservers-for-dns01-self-check
[cert-manager-uninstall]: https://cert-manager.io/docs/installation/uninstall/kubernetes