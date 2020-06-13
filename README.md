# Pipeline Restrictor

## About This Project

* Many Tekton pipelines use workspaces to exchange information between Tasks.
* These workspaces are bound to Persistent Volume Claims.
* This could cause problems when multiple PipelineRuns are active.
* The Pipeline Restrictor aims to get around this by ensuring that only a
single PipelineRun is active at a time.

## How It Works

* The Pipeline Restrictor is implemented as a Validating Admission Webhook
and is based off code from the
[OpenShift Kubernetes Namespace Reservation Webhook](https://github.com/openshift/kubernetes-namespace-reservation).
* Whenever a new PipelineRun object is created, the Pipeline Restrictor will
check to see if there are any PipelineRuns with the `pipelineRef` that are
currently running. If there are, Pipeline Restrictor will cancel those active
PipelineRuns before allowing the new PipelineRun to be created.

## Deployment Instructions

* Ensure that `oc` is in your `PATH`.
* Login as a user with admin privileges using `oc login`.
* Deploy the webhook server:

	```
	make webhook
	```
* This will do the following:
	* Kick-off a new s2i build.
	* Create an application based on the new imagestream.
	* Register the webhook server as an API Server Extension.
	* Register the API Server Extension as a Validating Admission Webhook.


## Notes

* The install script assumes that the pipelineruns reside in the `dev` project.
If that is not the case, don't forget to assign
`system:serviceaccount:pipelinerestrictor:default` the relevant permissions in
the target project.
* When deploying on OpenShift, a certificate is generated by the
[`service-ca` controller](https://docs.openshift.com/container-platform/4.4/authentication/certificates/service-serving-certificate.html#add-service-certificate_service-serving-certificate).
The web server is configured to use this certificate for SSL. The APIService
is also annotated so that the
[`service-ca` controller](https://docs.openshift.com/container-platform/4.4/authentication/certificates/service-serving-certificate.html#add-service-certificate-apiservice_service-serving-certificate)
injects the CA bundle into it. The generated certificate is stored in a secret
named `pipelinerestrictor`. The certificate needs to have the CN set to
`pipelinerestrictor.pipelinerestrictor.svc`. This is the hostname the kube API
server will use to connect to the webhook server.
* The webhook server mounts the secret as a volume.
* The webhook server is started with custom arguments to load the certificate
and key (`src/.s2i/bin/run`).
* When the webhook server starts up, it loads the client certificate details
from the `extension-apiserver-authentication` configmap in the `kube-system`
namespace.
* When you register the webhook server as an API server extension
(`yaml/apiserver.yaml`), the kube API server will proxy requests for the
specified resource (`kwkoo.github.com/v1beta1/requests`) to the webhook server.
When the kube API server connects to the webhook server,
	* It will need to recognize the webhook server's self-signed certificate -
	that's why `yaml/apiserver.yaml` is configured with `spec.caBundle` (which
	is the base64-encoded value of
	`apiserver.local.config/certificates/certificate.pem`).
	* It will also authenticate with the webhook server by presenting the
	client certificate from the `extension-apiserver-authentication` configmap.
* `yaml/webhook.yaml` registers the kube API server as a Validating Admission
Webhook. Whenever any resource is created, the kube API server will be invoked
with the validating resource set in `webhooks.clientConfig.service.path`. The
kube API server will then make a request to the webhook server.
* This is only for non-production deployments. As the generic-admission-server
README says, you need to implement a process for rotating the webhook
certificate if you want to deploy this in production.
* Some dependencies have problems compiling with Go 1.10 (which happens to be
the Go version in the latest SCL S2I image). This is why the build references a
non-SCL build image (which has Go 1.13).
* The service is set to listen on port 443 even though the webhook server
listens on port 8443. The reason for this is because the kube API server had
problems connecting (Jan 2020) to non-port 443 services even when a `port`
parameter was used in the Validating Admission config (back before the webhook
server was set to an API server extension).

## Troubleshooting

* In the event of a misconfiguration, a big source of issues is the kube API
server. To troubleshoot, look at the logs of the `kube-apiserver-XXX-master-0`
pods in the `openshift-kube-apiserver` project.
* When the webhook server comes up, it looks in the `kube-system` namespace
for the `extension-apiserver-authentication` configmap.
	* Turn on verbose logging in the webhook server
	(refer to `src/.s2i/bin/run`) and redeploy
	(`make cleanopenshift; sleep 30; make webhook`).
	* Look in the webhook server logs (`make logs`) to ensure that the webhook
	server manages to load the configmap.
* After the webhook server is registered as an
[API server extension](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/),
the kube API server should proxy requests for the validating resource
(`/apis/kwkoo.github.com/v1beta1/requests`) to the webhook server. To test,
`oc exec` into a running pod and execute the following:

	```
	curl \
	  -k \
	  -XPOST \
	  -H 'Authorization: Bearer TOKEN_FROM_OC_WHOAMI_-T' \
	  https://kubernetes.default.svc/apis/kwkoo.github.com/v1beta1/requests
	```
* If the kube API server is proxying requests properly, the `curl` above
should cause some output to be generated in the webhook server pod's logs (if
verbose logging is turned on).

## Go Dependencies

* The `generic-admission-server` has a dependency on the Prometheus project.
The Prometheus project has some dependencies which can only be loaded with
`go mod` - you will not be able to use `go get` to load the dependencies.
* Use the `generic-admission-server`'s
[`go.mod`](https://github.com/openshift/generic-admission-server/blob/master/go.mod)
file (together with all the tags) as a model. If you just enter
`github.com/openshift/generic-admission-server` as a dependency without any
tags, there will be a version conflict (as of 2020-01-18).
* If you wish to convert your project to the `vendor`-style, just do a
`go mod vendor` to transfer the dependencies into your project.