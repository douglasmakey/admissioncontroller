package deployments

import (
	"github.com/douglasmakey/admissioncontroller"

	"k8s.io/api/admission/v1beta1"
)

func validateDelete() admissioncontroller.AdmitFunc {
	return func(r *v1beta1.AdmissionRequest) (*admissioncontroller.Result, error) {
		dp, err := parseDeployment(r.OldObject.Raw)
		if err != nil {
			return &admissioncontroller.Result{Msg: err.Error()}, nil
		}

		if dp.Namespace == "special-system" && dp.Annotations["skip"] == "false" {
			return &admissioncontroller.Result{Msg: "You cannot remove a deployment from `special-system` namespace."}, nil
		}

		return &admissioncontroller.Result{Allowed: true}, nil
	}
}
