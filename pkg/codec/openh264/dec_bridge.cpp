#include "dec_bridge.hpp"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

void dec_trace(void* ctx, int level, const char* msg) {
    if (level == WELS_LOG_ERROR) {
        fprintf(stderr, "[openh264-dec] %s", msg);
    }
}

Decoder *dec_new(const DecoderOptions opts, int *eresult) {
  int rv;
  ISVCDecoder *engine;

  rv = WelsCreateDecoder(&engine);
  if (rv != 0) {
    *eresult = rv;
    return NULL;
  }

  SDecodingParam params = {0};
  params.uiTargetDqLayer = opts.target_dq_layer;
  params.eEcActiveIdc = opts.error_con_idc;
  params.bParseOnly = opts.parse_only;
  params.sVideoProperty.size = sizeof(SVideoProperty);
  params.sVideoProperty.eVideoBsType = opts.video_bs_type;

  int level = WELS_LOG_ERROR;
  engine->SetOption(DECODER_OPTION_TRACE_LEVEL, &level);
  engine->SetOption(DECODER_OPTION_TRACE_CALLBACK, (void*)dec_trace);
  engine->SetOption(DECODER_OPTION_TRACE_CALLBACK_CONTEXT, NULL);

  rv = engine->Initialize(&params);
  if (rv != 0) {
    *eresult = rv;
    return NULL;
  }

  Decoder *decoder = (Decoder *)malloc(sizeof(Decoder));
  decoder->engine = engine;
  decoder->params = params;
  decoder->buff = NULL;
  decoder->buff_size = 0;
  return decoder;
}

void dec_free(Decoder *d, int *eresult) {
  int rv = d->engine->Uninitialize();
  if (rv != 0) {
    *eresult = rv;
    return;
  }

  WelsDestroyDecoder(d->engine);

  if (d->buff) free(d->buff);
  free(d);
}

static DecodedFrame copy_frame(Decoder *d, SBufferInfo *info) {
  DecodedFrame frame = {0};

  int width = info->UsrData.sSystemBuffer.iWidth;
  int height = info->UsrData.sSystemBuffer.iHeight;
  int y_stride = info->UsrData.sSystemBuffer.iStride[0];
  int c_stride = info->UsrData.sSystemBuffer.iStride[1];

  int y_size = width * height;
  int c_size = (width/2) * (height / 2); // U and V
  int size = y_size + 2 * c_size;

  if (d->buff_size < size) {
    d->buff = (unsigned char *)malloc(size);
    d->buff_size = size;
  }

  unsigned char *y = d->buff;
  unsigned char *u = d->buff + y_size;
  unsigned char *v = d->buff + y_size + c_size;

  for (int i = 0; i < height; i++)
    memcpy(y + i * width, info->pDst[0] + i * y_stride, width);
  for (int i = 0; i < height / 2; i++)
    memcpy(u + i * (width/2), info->pDst[1] + i * c_stride, width / 2);
  for (int i = 0; i < height / 2; i++)
    memcpy(v + i * (width/2), info->pDst[2] + i * c_stride, width / 2);

  frame.y = y;
  frame.u = u;
  frame.v = v;
  frame.ystride = width;
  frame.cstride = width/2;
  frame.width = width;
  frame.height = height;
  frame.frame_ready = 1;
  return frame;
}

DecodedFrame dec_decode(Decoder *d, Slice s, int *eresult) {
  DecodedFrame frame = {0};
  SBufferInfo info = {0};

  int rv = d->engine->DecodeFrame2(s.data, s.data_len, info.pDst, &info);
  if (rv != 0) {
    *eresult = rv;
    return frame;
  }

  if (info.iBufferStatus != 1) {
    frame.frame_ready = 0;
    return frame;
  }

  return copy_frame(d, &info);
}

DecodedFrame dec_flush(Decoder *d, int *eresult) {
  DecodedFrame frame = {0};
  SBufferInfo info = {0};

  int rv = d->engine->FlushFrame(info.pDst, &info);
  if (rv != 0) {
    *eresult = rv;
    return frame;
  }

  if (info.iBufferStatus != 1) {
    frame.frame_ready = 0;
    return frame;
  }

  return copy_frame(d, &info);
}