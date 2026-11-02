//go:build live

package broker

import "errors"

// LiveBroker is the live-broker abstraction. It is deliberately isolated behind
// the live build tag and is absent from every default (paper-only) binary.
type LiveBroker interface {
	// RefuseLive reports that live trading is not enabled in this build.
	RefuseLive() error
}

// RefusingLiveBroker is the only LiveBroker implementation shipped behind the
// live build tag: it refuses every live operation.
type RefusingLiveBroker struct{}

// NewLiveBroker returns the refusing live-broker stub.
func NewLiveBroker() LiveBroker { return RefusingLiveBroker{} }

// NewRefusingLiveBroker returns the refusing live-broker stub.
func NewRefusingLiveBroker() *RefusingLiveBroker { return &RefusingLiveBroker{} }

// RefuseLive implements LiveBroker.
func (RefusingLiveBroker) RefuseLive() error {
	return errors.New("live broker is disabled; paper-only mode is enabled")
}

// RefusingLiveBroker implements LiveBroker.
func (RefusingLiveBroker) isLiveBroker() {}

// Compatibility aliases for the narrow live-broker surface.
type LiveBrokerClient = LiveBroker
type BrokerClient = LiveBroker
type RefusingBroker = RefusingLiveBroker
type RefusingBrokerClient = RefusingLiveBroker
type RefusingLiveBrokerClient = RefusingLiveBroker

// NewRefusingBroker returns the refusing live-broker stub.
func NewRefusingBroker() *RefusingBroker { return &RefusingLiveBroker{} }

// NewRefusingBrokerClient returns the refusing live-broker stub.
func NewRefusingBrokerClient() *RefusingBrokerClient { return &RefusingLiveBroker{} }

// NewRefusingLiveBrokerClient returns the refusing live-broker stub.
func NewRefusingLiveBrokerClient() *RefusingLiveBrokerClient { return &RefusingLiveBroker{} }
