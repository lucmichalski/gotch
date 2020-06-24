package nn

import (
	"github.com/sugarme/gotch"
	ts "github.com/sugarme/gotch/tensor"
)

type State interface{}

type RNN interface {

	// A zero state from which the recurrent network is usually initialized.
	ZeroState(batchDim int64) State

	// Applies a single step of the recurrent network.
	//
	// The input should have dimensions [batch_size, features].
	Step(input ts.Tensor, inState State) State

	// Applies multiple steps of the recurrent network.
	//
	// The input should have dimensions [batch_size, seq_len, features].
	// The initial state is the result of applying zero_state.
	Seq(input ts.Tensor) (ts.Tensor, State)

	// Applies multiple steps of the recurrent network.
	//
	// The input should have dimensions [batch_size, seq_len, features].
	SeqInit(input ts.Tensor, inState State) (ts.Tensor, State)
}

func defaultSeq(self interface{}, input ts.Tensor) (ts.Tensor, State) {
	batchDim := input.MustSize()[0]
	inState := self.(RNN).ZeroState(batchDim)

	return self.(RNN).SeqInit(input, inState)
}

// The state for a LSTM network, this contains two tensors.
type LSTMState struct {
	Tensor1 ts.Tensor
	Tensor2 ts.Tensor
}

// The hidden state vector, which is also the output of the LSTM.
func (ls LSTMState) H() (retVal ts.Tensor) {
	return ls.Tensor1.MustShallowClone()
}

// The cell state vector.
func (ls LSTMState) C() (retVal ts.Tensor) {
	return ls.Tensor2.MustShallowClone()
}

// The GRU and LSTM layers share the same config.
// Configuration for the GRU and LSTM layers.
type RNNConfig struct {
	HasBiases     bool
	NumLayers     int64
	Dropout       float64
	Train         bool
	Bidirectional bool
	BatchFirst    bool
}

// Default creates default RNN configuration
func DefaultRNNConfig() RNNConfig {
	return RNNConfig{
		HasBiases:     true,
		NumLayers:     1,
		Dropout:       float64(0.0),
		Train:         true,
		Bidirectional: false,
		BatchFirst:    true,
	}
}

// A Long Short-Term Memory (LSTM) layer.
//
// https://en.wikipedia.org/wiki/Long_short-term_memory
type LSTM struct {
	flatWeights []ts.Tensor
	hiddenDim   int64
	config      RNNConfig
	device      gotch.Device
}

// NewLSTM creates a LSTM layer.
func NewLSTM(vs *Path, inDim, hiddenDim int64, cfg RNNConfig) (retVal LSTM) {

	var numDirections int64 = 1
	if cfg.Bidirectional {
		numDirections = 2
	}

	gateDim := 4 * hiddenDim
	flatWeights := make([]ts.Tensor, 0)

	for i := 0; i < int(cfg.NumLayers); i++ {
		for n := 0; n < int(numDirections); n++ {
			if i != 0 {
				inDim = hiddenDim * numDirections
			}

			wIh := vs.KaimingUniform("w_ih", []int64{gateDim, inDim})
			wHh := vs.KaimingUniform("w_hh", []int64{gateDim, hiddenDim})
			bIh := vs.Zeros("b_ih", []int64{gateDim})
			bHh := vs.Zeros("b_hh", []int64{gateDim})

			flatWeights = append(flatWeights, wIh, wHh, bIh, bHh)
		}
	}

	return LSTM{
		flatWeights: flatWeights,
		hiddenDim:   hiddenDim,
		config:      cfg,
		device:      vs.Device(),
	}

}

// Implement RNN interface for LSTM:
// =================================

func (l LSTM) ZeroState(batchDim int64) (retVal State) {
	var numDirections int64 = 1
	if l.config.Bidirectional {
		numDirections = 2
	}

	layerDim := l.config.NumLayers * numDirections
	shape := []int64{layerDim, batchDim, l.hiddenDim}
	zeros := ts.MustZeros(shape, gotch.Float.CInt(), l.device.CInt())

	return LSTMState{
		Tensor1: zeros.MustShallowClone(),
		Tensor2: zeros.MustShallowClone(),
	}
}

func (l LSTM) Step(input ts.Tensor, inState State) (retVal State) {
	ip := input.MustUnsqueeze(1, false)

	_, state := l.SeqInit(ip, inState.(LSTMState))

	return state
}

func (l LSTM) Seq(input ts.Tensor) (ts.Tensor, State) {
	return defaultSeq(l, input)
}

func (l LSTM) SeqInit(input ts.Tensor, inState State) (ts.Tensor, State) {

	output, h, c := input.MustLSTM([]ts.Tensor{inState.(LSTMState).Tensor1, inState.(LSTMState).Tensor2}, l.flatWeights, l.config.HasBiases, l.config.NumLayers, l.config.Dropout, l.config.Train, l.config.Bidirectional, l.config.BatchFirst)

	return output, LSTMState{
		Tensor1: h,
		Tensor2: c,
	}
}

// GRUState is a GRU state. It contains a single tensor.
type GRUState struct {
	Tensor ts.Tensor
}

func (gs GRUState) Value() ts.Tensor {
	return gs.Tensor
}

// A Gated Recurrent Unit (GRU) layer.
//
// https://en.wikipedia.org/wiki/Gated_recurrent_unit
type GRU struct {
	flatWeights []ts.Tensor
	hiddenDim   int64
	config      RNNConfig
	device      gotch.Device
}

// NewGRU create a new GRU layer
func NewGRU(vs *Path, inDim, hiddenDim int64, cfg RNNConfig) (retVal GRU) {
	var numDirections int64 = 1
	if cfg.Bidirectional {
		numDirections = 2
	}

	gateDim := 3 * hiddenDim
	flatWeights := make([]ts.Tensor, 0)

	for i := 0; i < int(cfg.NumLayers); i++ {
		for n := 0; n < int(numDirections); n++ {
			if i != 0 {
				inDim = hiddenDim * numDirections
			}

			wIh := vs.KaimingUniform("w_ih", []int64{gateDim, inDim})
			wHh := vs.KaimingUniform("w_hh", []int64{gateDim, hiddenDim})
			bIh := vs.Zeros("b_ih", []int64{gateDim})
			bHh := vs.Zeros("b_hh", []int64{gateDim})

			flatWeights = append(flatWeights, wIh, wHh, bIh, bHh)
		}
	}

	return GRU{
		flatWeights: flatWeights,
		hiddenDim:   hiddenDim,
		config:      cfg,
		device:      vs.Device(),
	}
}

// Implement RNN interface for GRU:
// ================================

func (g GRU) ZeroState(batchDim int64) (retVal State) {
	var numDirections int64 = 1
	if g.config.Bidirectional {
		numDirections = 2
	}

	layerDim := g.config.NumLayers * numDirections
	shape := []int64{layerDim, batchDim, g.hiddenDim}

	return ts.MustZeros(shape, gotch.Float.CInt(), g.device.CInt())
}

func (g GRU) Step(input ts.Tensor, inState State) (retVal State) {
	ip := input.MustUnsqueeze(1, false)

	_, state := g.SeqInit(ip, inState.(LSTMState))

	return state
}

func (g GRU) Seq(input ts.Tensor) (ts.Tensor, State) {
	return defaultSeq(g, input)
}

func (g GRU) SeqInit(input ts.Tensor, inState State) (ts.Tensor, State) {

	output, h := input.MustGRU(inState.(GRUState).Tensor, g.flatWeights, g.config.HasBiases, g.config.NumLayers, g.config.Dropout, g.config.Train, g.config.Bidirectional, g.config.BatchFirst)

	return output, GRUState{Tensor: h}
}