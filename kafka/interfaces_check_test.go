package kafka

// Compile-time checks that concrete types implement their service interfaces.
var (
	_ ConsumerService[any] = (*consumerImpl[any])(nil)
	_ ProducerService[any] = (*producerImpl[any])(nil)
	_ Pingable             = (*consumerImpl[any])(nil)
	_ Pingable             = (*producerImpl[any])(nil)
	_ Lifetime             = (*consumerImpl[any])(nil)
	_ Lifetime             = (*producerImpl[any])(nil)
)
