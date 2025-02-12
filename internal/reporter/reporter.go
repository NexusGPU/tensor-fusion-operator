package reporter

import (
	"context"
	"fmt"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

type Reporter interface {
	Report(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error
}

type DryRunReporter struct {
}

func NewDryRunReporter() Reporter {
	return &DryRunReporter{}
}

func (r *DryRunReporter) Report(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error {
	log := log.FromContext(ctx)
	if err := f(); err != nil {
		return err
	}
	objYaml, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	log.Info("\n" + string(objYaml))
	return nil
}

type KubeReporter struct {
	client    client.Client
	namespace string
}

func NewKubeReporter(namespace string) (Reporter, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("find cluster kubeConfig %w", err)
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(tfv1.AddToScheme(scheme))
	client, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("create kubeClient %w", err)
	}

	return &KubeReporter{
		client,
		namespace,
	}, nil
}

func (r *KubeReporter) Report(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error {
	_, err := controllerutil.CreateOrUpdate(ctx, r.client, obj, f)
	return err
}
