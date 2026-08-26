package openh264

// #include <string.h>
// #include <openh264/codec_api.h>
// #include <errno.h>
// #include "dec_bridge.hpp"
import "C"

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"sync"
	"unsafe"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/prop"
)

type decoder struct {
	engine *C.Decoder
	r      io.Reader
	buf    []byte

	mu     sync.Mutex
	closed bool
}

func NewDecoder(r io.Reader, p prop.Media, params DecParams) (codec.VideoDecoder, error) {
	if params.BitRate == 0 {
		params.BitRate = 100000
	}

	var rv C.int
	cdecoder := C.dec_new(C.DecoderOptions{
		target_dq_layer: C.uchar(params.TargetDQLayer),
		error_con_idc:   C.ERROR_CON_IDC(C.uint(params.ErrorConcealment)),
		parse_only:      C.bool(params.ParseOnly),
		video_bs_type:   C.VIDEO_BITSTREAM_TYPE(C.uint(params.VideoBitstreamType)),
	}, &rv)
	if err := errResult(rv); err != nil {
		return nil, fmt.Errorf("failed in creating decoder: %v", err)
	}

	return &decoder{
		engine: cdecoder,
		r:      r,
		buf:    make([]byte, 1024*1024),
	}, nil
}

func toCSlice(b []byte) C.Slice {
	var s C.Slice
	if len(b) > 0 {
		s.data = (*C.uchar)(unsafe.Pointer(&b[0]))
		s.data_len = C.int(len(b))
	} else {
		s.data = nil
		s.data_len = 0
	}
	return s
}

func (d *decoder) Read() (image.Image, func(), error) {
	var (
		eresult  C.int
		efresult C.int
		dst      *image.YCbCr
		frame    C.DecodedFrame
	)

	// frame = C.dec_decode(d.engine, toCSlice(d.buf), &eresult)
	// if eresult != 0 {
	// 	return nil, nil, fmt.Errorf("decode error: %d", int(eresult))
	// }
	// if frame.frame_ready != 0 {
	// 	dst = processFrame(frame)
	// 	return dst, func() {}, nil
	// }

	for {
		n, err := d.r.Read(d.buf)
		log.Printf("first bytes: %x", d.buf[:min(20, n)])
		if err != nil {
			if errors.Is(err, io.EOF) {
				for {
					frame = C.dec_flush(d.engine, &efresult)
					if efresult != 0 {
						return nil, nil, fmt.Errorf("flush error: %d", int(efresult))
					}
					if frame.frame_ready == 0 {
						return nil, nil, io.EOF
					}
					dst = processFrame(frame)
					return dst, func() {}, nil
				}
			}

			return nil, nil, err
		}

		frame = C.dec_decode(d.engine, toCSlice(d.buf[:n]), &eresult)
		if eresult != 0 && efresult != C.int(0x01) {
			return nil, nil, fmt.Errorf("decode error: %d", int(eresult))
		}
		if frame.frame_ready != 0 {
			dst = processFrame(frame)
			return dst, func() {}, nil
		}

	}
}

func processFrame(frame C.DecodedFrame) *image.YCbCr {
	w := int(frame.width)
	h := int(frame.height)

	ySrc := unsafe.Slice((*byte)(unsafe.Pointer(frame.y)), w*h)
	uSrc := unsafe.Slice((*byte)(unsafe.Pointer(frame.u)), w/2*h/2)
	vSrc := unsafe.Slice((*byte)(unsafe.Pointer(frame.v)), w/2*h/2)

	dst := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)

	copy(dst.Y, ySrc)
	copy(dst.Cb, uSrc)
	copy(dst.Cr, vSrc)

	return dst
}

func (d *decoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true

	var rv C.int
	C.dec_free(d.engine, &rv)
	return errResult(rv)
}
