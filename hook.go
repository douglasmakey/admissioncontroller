package admissioncontroller

import (
	"fmt"

	admission "k8s.io/api/admission/v1beta1"
)

// Result contains the result of an admission request
type Result struct {
	Allowed  bool
	Msg      string
	PatchOps []PatchOperation
}

// AdmitFunc defines how to process an admission request
type AdmitFunc func(request *admission.AdmissionRequest) (*Result, error)

// Hook represents the set of functions for each operation in an admission webhook.
type Hook struct {
	Create  AdmitFunc
	Delete  AdmitFunc
	Update  AdmitFunc
	Connect AdmitFunc
}

// Execute evaluates the request and try to execute the function for operation specified in the request.
func (h *Hook) Execute(r *admission.AdmissionRequest) (*Result, error) {
	switch r.Operation {
	case admission.Create:
		return wrapperExecution(h.Create, r)
	case admission.Update:
		return wrapperExecution(h.Update, r)
	case admission.Delete:
		return wrapperExecution(h.Delete, r)
	case admission.Connect:
		return wrapperExecution(h.Connect, r)
	}

	return &Result{Msg: fmt.Sprintf("Invalid operation: %s", r.Operation)}, nil
}

func wrapperExecution(fn AdmitFunc, r *admission.AdmissionRequest) (*Result, error) {
	if fn == nil {
		return nil, fmt.Errorf("operation %s is not registered", r.Operation)
	}

	return fn(r)
}
