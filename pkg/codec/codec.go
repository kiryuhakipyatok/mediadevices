package codec

import (
	"image"
	"io"

	"github.com/pion/mediadevices/pkg/prop"
)

// ReadCloser is an io.ReadCloser with a controller
type ReadCloser interface {
	Read() (b []byte, release func(), err error)
	Close() error
	Controllable
}

type VideoDecoderBuilder interface {
	BuildVideoDecoder(r io.Reader, p prop.Media) (VideoDecoder, error)
}

type VideoDecoder interface {
	Read() (image.Image, func(), error)
	Close() error
}

// EncoderController is the interface allowing to control the encoder behaviour after it's initialisation.
// It will possibly have common control method in the future.
// A controller can have optional methods represented by *Controller interfaces
type EncoderController any

// Controllable is a interface representing a encoder which can be controlled
// after it's initialisation with an EncoderController
type Controllable interface {
	Controller() EncoderController
}

// KeyFrameController is a interface representing an encoder that can be forced to produce key frame on demand
type KeyFrameController interface {
	EncoderController
	// ForceKeyFrame forces the next frame to be a keyframe, aka intra-frame.
	ForceKeyFrame() error
}

// BitRateController is a interface representing an encoder which can have a variable bit rate
type BitRateController interface {
	EncoderController
	// SetBitRate sets current target bitrate, lower bitrate means smaller data will be transmitted
	// but this also means that the quality will also be lower.
	SetBitRate(int) error
}

type QPController interface {
	EncoderController
	// DynamicQPControl adjusts the QP of the encoder based on the current and target bitrate
	DynamicQPControl(currentBitrate int, targetBitrate int) error
}

// BaseParams represents an codec's encoding properties
type BaseParams struct {
	// Target bitrate in bps.
	BitRate int

	// Expected interval of the keyframes in frames.
	KeyFrameInterval int
}
