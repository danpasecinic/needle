package needle

func New(opts ...Option) *Container {
	return newContainer(opts...)
}
