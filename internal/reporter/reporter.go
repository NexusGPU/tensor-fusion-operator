package reporter

import (
	"context"
	"fmt"
	"os"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(tfv1.AddToScheme(Scheme))
}

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
	kubeConfigEnvVar := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
	if kubeConfigEnvVar != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigEnvVar)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("find cluster kubeConfig %w", err)
	}

	client, err := client.New(config, client.Options{
		Scheme: Scheme,
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
	statusObj := obj.DeepCopyObject().(client.Object)
	_, err := controllerutil.CreateOrUpdate(ctx, r.client, obj, func() error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("create or update err: %w", err)
	}

	if err := f(); err != nil {
		return err
	}
	// available rewrite to wrong value
	return r.client.Status().Update(ctx, statusObj)
}
