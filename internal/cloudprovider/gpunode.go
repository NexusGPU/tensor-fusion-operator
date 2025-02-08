package cloudprovider

// TODO
type GPUNodeProvider interface {
	Create(identifier string) error
	Terminate(identifier string) error
}
