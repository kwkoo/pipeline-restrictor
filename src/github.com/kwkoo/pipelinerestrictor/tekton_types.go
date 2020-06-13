package pipelinerestrictor

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PipelineRunGVR returns the Group Version Resource of a PipelineRun.
func PipelineRunGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "pipelineruns",
	}
}

// PipelineRunFromUnstructured converts an Unstructured object to a
// PipelineRun.
func PipelineRunFromUnstructured(in unstructured.Unstructured) (PipelineRun, error) {
	pr := PipelineRun{}
	o := in.Object
	if metadataRaw, ok := o["metadata"]; ok {
		if metadata, ok := metadataRaw.(map[string]interface{}); ok {
			if gnRaw, ok := metadata["generateName"]; ok {
				if gn, ok := gnRaw.(string); ok {
					pr.Metadata.GenerateName = gn
				}
			}
			if nRaw, ok := metadata["name"]; ok {
				if n, ok := nRaw.(string); ok {
					pr.Metadata.Name = n
				}
			}
		}
	}
	if specRaw, ok := o["spec"]; ok {
		if spec, ok := specRaw.(map[string]interface{}); ok {
			if prRefRaw, ok := spec["pipelineRef"]; ok {
				if prRef, ok := prRefRaw.(map[string]interface{}); ok {
					pr.Spec.PipelineRef = &PipelineRef{}
					if nRaw, ok := prRef["name"]; ok {
						if n, ok := nRaw.(string); ok {
							pr.Spec.PipelineRef.Name = n
						}
					}
				}
			}
			if statusRaw, ok := spec["status"]; ok {
				if status, ok := statusRaw.(string); ok {
					pr.Spec.Status = status
				}
			}
		}
	}
	if statusRaw, ok := o["status"]; ok {
		if status, ok := statusRaw.(map[string]interface{}); ok {
			if condsRaw, ok := status["conditions"]; ok {
				if conds, ok := condsRaw.([]interface{}); ok {
					pr.Status.Conditions = []Condition{}
					for _, cRaw := range conds {
						condition := Condition{}
						c, ok := cRaw.(map[string]interface{})
						if !ok {
							continue
						}
						if tRaw, ok := c["type"]; ok {
							if t, ok := tRaw.(string); ok {
								condition.Type = t
							}
						}
						if cstatRaw, ok := c["status"]; ok {
							if cstat, ok := cstatRaw.(string); ok {
								condition.Status = cstat
							}
						}
						if sevRaw, ok := c["severity"]; ok {
							if sev, ok := sevRaw.(string); ok {
								condition.Severity = sev
							}
						}
						if reasonRaw, ok := c["reason"]; ok {
							if reason, ok := reasonRaw.(string); ok {
								condition.Reason = reason
							}
						}
						if messageRaw, ok := c["message"]; ok {
							if message, ok := messageRaw.(string); ok {
								condition.Message = message
							}
						}
						// add status, severity, reason, message
						pr.Status.Conditions = append(pr.Status.Conditions, condition)
					}
				}
			}
		}
	}

	return pr, nil
}

// PipelineRun represents a single execution of a Pipeline. PipelineRuns are how
// the graph of Tasks declared in a Pipeline are executed; they specify inputs
// to Pipelines such as parameter values and capture operational aspects of the
// Tasks execution such as service account and tolerations. Creating a
// PipelineRun creates TaskRuns for Tasks in the referenced Pipeline.
type PipelineRun struct {
	Metadata struct {
		GenerateName string `json:"generateName,omitempty"`
		Name         string `json:"name"`
	} `json:"metadata,omitempty"`
	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

// GetName returns the PipelineRun's name.
func (pr PipelineRun) GetName() string {
	return pr.Metadata.Name
}

// IsDone returns true if the PipelineRun's status indicates that it is done.
func (pr *PipelineRun) IsDone() bool {
	for _, c := range pr.Status.Conditions {
		if c.Type == "Succeeded" {
			return !(c.Status == "Unknown")
		}

	}
	return true
}

// IsCancelled returns true if the PipelineRun's spec status is set to Cancelled state
func (pr *PipelineRun) IsCancelled() bool {
	return pr.Spec.Status == "PipelineRunCancelled"
}

// PipelineRunSpec defines the desired state of PipelineRun
type PipelineRunSpec struct {
	// +optional
	PipelineRef *PipelineRef `json:"pipelineRef,omitempty"`
	// Used for cancelling a pipelinerun (and maybe more later on)
	// +optional
	Status string `json:"status,omitempty"`
}

// PipelineRef can be used to refer to a specific instance of a Pipeline.
type PipelineRef struct {
	Name string
}

// PipelineRunStatus defines the observed state of PipelineRun
type PipelineRunStatus struct {
	Status `json:",inline"`

	// PipelineRunStatusFields inlines the status fields.
	//PipelineRunStatusFields `json:",inline"`
}

// Conditions is a simple wrapper around apis.Conditions to implement duck.Implementable.
type Conditions []Condition

// Status shows how we expect folks to embed Conditions in
// their Status field.
// WARNING: Adding fields to this struct will add them to all Knative resources.
type Status struct {
	// ObservedGeneration is the 'Generation' of the Service that
	// was last processed by the controller.
	// +optional
	//ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions the latest available observations of a resource's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// Condition defines a readiness condition for a Knative resource.
// See: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
// +k8s:deepcopy-gen=true
type Condition struct {
	// Type of condition.
	// +required
	Type string `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status string `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// Severity with which to treat failures of this type of condition.
	// When this is not specified, it defaults to Error.
	// +optional
	Severity string `json:"severity,omitempty" description:"how to interpret failures of this condition, one of Error, Warning, Info"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// IsTrue is true if the condition is True
func (c *Condition) IsTrue() bool {
	if c == nil {
		return false
	}
	return c.Status == "True"
}

// IsFalse is true if the condition is False
func (c *Condition) IsFalse() bool {
	if c == nil {
		return false
	}
	return c.Status == "False"
}

// IsUnknown is true if the condition is Unknown
func (c *Condition) IsUnknown() bool {
	if c == nil {
		return true
	}
	return c.Status == "Unknown"
}
