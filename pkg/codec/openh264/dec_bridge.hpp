#pragma once

#include <openh264/codec_api.h>
#include "enc_bridge.hpp"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct DecodedFrame {
  unsigned char *y, *u, *v;
  int ystride;
  int cstride;
  int width;
  int height;
  int frame_ready;
} DecodedFrame;

typedef struct DecoderOptions {
  unsigned char target_dq_layer;
  ERROR_CON_IDC error_con_idc;
  bool parse_only;
  VIDEO_BITSTREAM_TYPE video_bs_type;
} DecoderOptions;

typedef struct Decoder {
  SDecodingParam params;
  ISVCDecoder *engine;
  unsigned char *buff;
  int buff_size;
} Decoder;

Decoder *dec_new(const DecoderOptions opts, int *eresult);
void dec_free(Decoder *d, int *eresult);
DecodedFrame dec_decode(Decoder *d, Slice s, int *eresult);
DecodedFrame dec_flush(Decoder *d, int *eresult);
#ifdef __cplusplus
}
#endif