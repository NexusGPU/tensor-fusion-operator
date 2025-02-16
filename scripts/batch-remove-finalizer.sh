for resource in $(kubectl get gpunodes -o jsonpath='{.items[*].metadata.name}'); do
  kubectl patch gpunode $resource --type=merge -p '{"metadata":{"finalizers":[]}}'
done