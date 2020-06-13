package pipelinerestrictor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const timeout = 5

// AdmissionHook is a struct that contains data needed for the admission
// webhook.
type AdmissionHook struct {
	client dynamic.Interface
}

// CancelActiveRuns cancels all active PipelineRuns that have the given
// pipelineRef.
func (a *AdmissionHook) CancelActiveRuns(namespace, pipelineRef string) (int, error) {
	if len(pipelineRef) == 0 {
		return 0, errors.New("called with empty pipelineRef")
	}
	list, err := a.listPipelineRuns(namespace)
	if err != nil {
		return 0, fmt.Errorf("error getting pipelineruns: %v", err)
	}
	cancelledCount := 0
	for _, item := range list.Items {
		pr, err := PipelineRunFromUnstructured(item)
		if err != nil {
			log.Printf("skipping pipelinerun because we could not parse it: %v", err)
			continue
		}
		if pr.IsDone() || pr.IsCancelled() {
			continue
		}
		prName := pr.Metadata.Name
		if len(prName) == 0 {
			continue
		}
		if pr.Spec.PipelineRef == nil {
			continue
		}
		pRef := pr.Spec.PipelineRef.Name
		if len(pRef) == 0 || pRef != pipelineRef {
			continue
		}
		if err := a.cancelPipelineRun(namespace, prName); err != nil {
			log.Printf("error cancelling pipelinerun: %v", err)
			continue
		}
		log.Printf("successfully cancelled pipelinerun %s", prName)
		cancelledCount++
	}
	return cancelledCount, nil
}

func (a *AdmissionHook) listPipelineRuns(namespace string) (*unstructured.UnstructuredList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	return a.client.Resource(PipelineRunGVR()).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

func (a *AdmissionHook) cancelPipelineRun(namespace, name string) error {
	payload := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: "PipelineRunCancelled",
	}}
	data, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	_, err := a.client.Resource(PipelineRunGVR()).Namespace(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	return err
}

// ValidatingResource is the GVR for the webhook.
func (a *AdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "kwkoo.github.com",
			Version:  "v1beta1",
			Resource: "requests",
		},
		"request"
}

// Validate is called when a new PipelineRun object is created.
func (a *AdmissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}
	status.Allowed = true

	//log.Print("Operation:", admissionSpec.Operation)
	namespace := admissionSpec.Namespace
	if len(namespace) == 0 {
		log.Print("could not get namespace, skipping checks")
		return status
	}

	run := &PipelineRun{}
	if err := json.Unmarshal(admissionSpec.Object.Raw, run); err != nil {
		log.Print("could not unmarshal raw object")
		return status
	}
	if run.Spec.PipelineRef == nil || len(run.Spec.PipelineRef.Name) == 0 {
		log.Print("could not get pipelineRef name, skipping checks")
		return status
	}
	log.Printf("create %s in %s with pipelineref %s", admissionSpec.Name, namespace, run.Spec.PipelineRef.Name)

	cancelledCount, err := a.CancelActiveRuns(namespace, run.Spec.PipelineRef.Name)
	if err != nil {
		status.Result = &metav1.Status{Message: fmt.Sprintf("Encountered error while trying to cancel active runs: %v", err)}
		return status
	}
	status.Result = &metav1.Status{Message: fmt.Sprintf("Cancelled %d pipelinerun(s)", cancelledCount)}
	/*
		if cancelledCount > 0 {
			status.Allowed = false
		}
	*/
	return status
}

// Initialize sets up the k8s client.
func (a *AdmissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	d, err := dynamic.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	a.client = d
	return nil
}
