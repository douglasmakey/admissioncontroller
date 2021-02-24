# Go admission controller

This repository aims to show you a basic boilerplate of an admission controller in go.

[Kubernetes admission controllers](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers)

> In a nutshell, Kubernetes admission controllers are plugins that govern and enforce how the cluster is used. They can 
> be thought of as a gatekeeper that intercept (authenticated) API requests and may change the request object or deny the request altogether.
> The admission control process has two phases: the mutating phase is executed first, followed by the validating phase.

Kubernetes admission Controller Phases:

![](https://d33wubrfki0l68.cloudfront.net/af21ecd38ec67b3d81c1b762221b4ac777fcf02d/7c60e/images/blog/2019-03-21-a-guide-to-kubernetes-admission-controllers/admission-controller-phases.png)

## Core

At the root level, we have defined some core structs.

In `admission.go` we have the main structs: `AdmitFunc` , `Hook`, and `Result`.

`AdmitFunc` is a function type that defines how to process an admission request. It is where you define
the validations or mutations for a specific request. You will see some examples in `deployments` and `pods` packages.

```go
type AdmitFunc func(request *admission.AdmissionRequest) (*Result, error)

```

`Hook` is representing the set of functions (`AdmitFunc`) for each operation in an admission webhook. When you create an admission 
webhook, either validating or mutating, you have to define which operations you want to intervene.

```go
type Hook struct {
	Create  AdmitFunc
	Delete  AdmitFunc
	Update  AdmitFunc
	Connect AdmitFunc
}
```

For example, you might want to create a validation webhook to apply a specific validation in the pods' creation. 
For that, you have to create a `ValidatingWebhookConfiguration` as the following:

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
...
webhooks:
  - name: admission-server.default.svc
    clientConfig:
      service:
        ...
    rules:
      - operations: ["CREATE"] # which operations you want to match.
        ...
        resources: ["pods"]
```

So, now you can create a `Hook` instance for that webhook, just setting the `Create` function. If your webhook handles 
more operations, you should create the functions and set them for each operation.

You can see a better example in the `deployments` package.

```go
// webhook with just one operation [CREATE]
hook := admissioncontroller.Hook{Create: myValidationFunction}

// webhook with multiple operations [CREATE,DELETE]
hook := admissioncontroller.Hook{Create: createValidation, Delete: deleteValidation}
```

In `patch.go` we have the struct and function for JSON patch operation.

`PatchOperation` represents a JSON patch operation.

A mutating admission webhook may modify the incoming object in the request. This is done using the JSON patch format. 
See JSON patch documentation for more details.

You can see a better example in the function `mutateCreate` inside the `pods` package, where we use `PatchOperation` to set an annotation to the pod and
also, to add a sidecar container.

```go
type PatchOperation struct {
	Op    string
	Path  string
	From  string
	Value interface{}
}
```


## Packages

### pods

This package should contain all the validations and mutations for `Pods` resources.

For example, we have a function to reject a pod creation request if any of the pod's containers are using the latest tag.

```go
func validateCreate() admissioncontroller.AdmitFunc {
	return func(r *v1beta1.AdmissionRequest) (*admissioncontroller.Result, error) {
		pod, err := parsePod(r.Object.Raw)
		if err != nil {
			return &admissioncontroller.Result{Msg: err.Error()}, nil
		}
		
		for _, c := range pod.Spec.Containers {
			if strings.HasSuffix(c.Image, ":latest") {
				return &admissioncontroller.Result{Msg: "You cannot use the tag 'latest' in a container."}, nil
			}
		}
		
		return &admissioncontroller.Result{Allowed: true}, nil
	}
}
```

Also, we have the function `mutateCreate`, which is used in a `MutatinWebhook`, this function uses `PatchOperations` to 
tell  Kubernetes that have to make certain modifications to the pod creation, the mutations are:

* Using `JSON Patch` operation to add an annotations to the pod.
* Using `JSON Patch` operation to replace the pod's containers adding a new container as a simple sidecar container. This
  is such a powerful feature. For example, Istio uses a similar approach to inject its sidecar containers into each pod.

```go
func mutateCreate() admissioncontroller.AdmitFunc {
	return func(r *v1beta1.AdmissionRequest) (*admissioncontroller.Result, error) {
		var operations []admissioncontroller.PatchOperation
		// ...
		if pod.Namespace == "special" {
			var containers []v1.Container
			containers = append(containers, pod.Spec.Containers...)
			sideC := v1.Container{
				Name:    "test-sidecar",
				Image:   "busybox:stable",
				Command: []string{"sh", "-c", "while true; do echo 'I am a container injected by mutating webhook'; sleep 2; done"},
			}
			containers = append(containers, sideC)
			operations = append(operations, admissioncontroller.ReplacePatchOperation("/spec/containers", containers))
		}
		
		metadata := map[string]string{"origin": "fromMutation"}
		operations = append(operations, admissioncontroller.AddPatchOperation("/metadata/annotations", metadata))
		return &admissioncontroller.Result{
			Allowed:  true,
			PatchOps: operations,
		}, nil
	}
}
```

The idea is that you can have a different package to handle one or multiple resources.

For example, you could have an `annotations` package to mutate or validate annotations cross resources such as `Pod`,
`Deployments`, `DaemonSet`, etc.

### deployments

This package should contain all the validations and mutations for `Deployments` resources.

The current examples are:
* The `validateCreate` function validates in a create operation if the deployment namespace is `special`. If it is, the
  function will reject the request.
* The `validateDelete` function validates in a delete operation if the deployment namespace is `special-system`. If it is,
  the function will reject the request.

### http

Contains the http server and its handlers.

`http.NewServer` returns an HTTP server, and here we will register all our webhooks.

```go
// NewServer creates and return a http.Server
func NewServer(port string) *http.Server {
	// Instances hooks
	podsValidation := pods.NewValidationHook()
	deploymentValidation := deployments.NewValidationHook()
	//....
	// Routers
	ah := newAdmissionHandler()
	//...
	mux.Handle("/validate/pods", ah.Serve(podsValidation))
	mux.Handle("/validate/deployments", ah.Serve(deploymentValidation))
	
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}
}
```

`admissionHandler` represents the HTTP handler for an admission webhook.

```go
type admissionHandler struct {
	decoder runtime.Decoder
}

// Serve returns an http.HandlerFunc for an admission webhook that contains all the
// logic to process an admission webhook request.
func (h *admissionHandler) Serve(hook admissioncontroller.Hook) http.HandlerFunc {
	//...
}
```
### demo

Contains all the files required to run a demo of this admission controller. Using the `deploy.sh` script, you can deploy
the admission controller in a k8s cluster.

**Note: `demo/deploy.sh` is just for develop/test environment. It was not intended for production.**

A cluster on which this example can be tested must be running Kubernetes 1.9.0 or above. The cluster should have
the admissionregistration.k8s.io/v1beta1 API enabled. You can verify that using the following command:

```
kubectl api-versions
...
admissionregistration.k8s.io/v1beta1
...
```

If you want to use a mutation admission, the `MutatingAdmissionWebhook` admission controller should be added in the 
admission-control flag of the kube-apiserver.

You can check which admission controllers are activated inspecting the `kube-apiserver`

```text
--enable-admission-plugins=..,MutatingAdmissionWebhook,ValidatingAdmissionWebhook.."
```

Run `demo/deploy.sh` will create a self-signed CA, a certificate and private for the server and the webhooks, also
will create the following resources: `tls secret`, `Deployment`, and all the `Admission webhooks`.

You can see all the created resources:

```shell
kubectl get deployment
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
admission-server   1/1     0            1           1h

kubectl get secret
NAME                  TYPE                                  DATA   AGE
admission-tls         kubernetes.io/tls                     2      1h

kubectl get mutatingwebhookconfigurations
NAME           WEBHOOKS   AGE
pod-mutation   1          1h

kubectl get validatingwebhookconfigurations
NAME                    WEBHOOKS   AGE
deployment-validation   1          1h
pod-validation          1          1h
```

Then we can use the different manifests inside `demo/pods` and `demo/deployments` to test the validations and mutations
that we have registered.
