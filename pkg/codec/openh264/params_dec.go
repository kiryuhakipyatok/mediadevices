package openh264

// #include <openh264/codec_api.h>
import "C"

import (
	"io"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/prop"
)

type DecParams struct {
	codec.BaseParams
	TargetDQLayer      TargetDQLayer
	VideoBitstreamType VideoBSType
	ErrorConcealment   ErrorConcealmentIDC
	ParseOnly          bool
}

type TargetDQLayer uint

const (
	TargetDQLayerAll  = 0xFF
	TargetDQLayerBase = 0
)

type VideoBSType uint

const (
	VideoBitstreamAVC     = C.VIDEO_BITSTREAM_AVC
	VideoBitstreamSVC     = C.VIDEO_BITSTREAM_SVC
	VideoBitstreamDefault = C.VIDEO_BITSTREAM_DEFAULT
)

type ErrorConcealmentIDC uint

const (
	ErrorConDisable                           = C.ERROR_CON_DISABLE
	ErrorConFrameCopy                         = C.ERROR_CON_FRAME_COPY
	ErrorConSliceCopu                         = C.ERROR_CON_SLICE_COPY
	ErrorConSliceCopyCrossIDR                 = C.ERROR_CON_SLICE_COPY_CROSS_IDR
	ErrorConSliceCopyCrossIDRFreezeResChange  = C.ERROR_CON_SLICE_COPY_CROSS_IDR_FREEZE_RES_CHANGE
	ErrorConSliceMvCopyCrossIDR               = C.ERROR_CON_SLICE_MV_COPY_CROSS_IDR
	ErrorConSliceMvCopyCrossIDRFeezeResChange = C.ERROR_CON_SLICE_MV_COPY_CROSS_IDR_FREEZE_RES_CHANGE
)

func NewDecParams() (DecParams, error) {
	return DecParams{
		BaseParams: codec.BaseParams{
			BitRate: 100000,
		}, 
		TargetDQLayer:      TargetDQLayerAll,
		VideoBitstreamType: VideoBitstreamDefault,
		ErrorConcealment:   ErrorConFrameCopy,
	}, nil
}

func (p *DecParams) BuildVideoDecoder(r io.Reader, property prop.Media) (codec.VideoDecoder, error) {
	return NewDecoder(r, property, *p)
}
