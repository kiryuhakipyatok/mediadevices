package screen

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
	"golang.org/x/image/draw"
)

type Screen struct {
	num                   int
	reader                *reader
	tick                  *time.Ticker
	imgBuffPool           sync.Pool
	downscaledImgBuffPool sync.Pool
}

func deviceID(num int) string {
	return fmt.Sprintf("X11Screen%d", num)
}

func init() {
	Initialize()
}

// Initialize finds and registers active displays. This is part of an experimental API.
func Initialize() {
	dp, err := openDisplay()
	if err != nil {
		// No x11 display available.
		return
	}
	defer dp.Close()
	numScreen := dp.NumScreen()
	for i := 0; i < numScreen; i++ {
		s := NewScreen(i)
		driver.GetManager().Register(s,
			driver.Info{
				Label:      deviceID(i),
				DeviceType: driver.Screen,
			},
		)
	}
}

func NewScreen(screenNum int) *Screen {
	s := Screen{
		num: screenNum,
	}
	return &s
}

func (s *Screen) Open() error {
	r, err := newReader(s.num)
	if err != nil {
		return err
	}
	s.reader = r
	return nil
}

func (s *Screen) Close() error {
	s.reader.Close()
	if s.tick != nil {
		s.tick.Stop()
		s.tick = nil
	}
	return nil
}

func (s *Screen) GetCaptureName() string {
	return "X11"
}

func (s *Screen) VideoRecord(p prop.Media) (video.Reader, error) {
	if p.FrameRate == 0 {
		p.FrameRate = 10
	}
	if p.Width == 0 {
		p.Width = 1920
	}
	if p.Height == 0 {
		p.Height = 1080
	}
	screenProp := s.Properties()[0]
	isDiff := p.Width == screenProp.Width && p.Height == screenProp.Height
	s.tick = time.NewTicker(time.Duration(float32(time.Second) / p.FrameRate))

	screenBounds := image.Rect(0, 0, screenProp.Width, screenProp.Height)
	s.imgBuffPool = sync.Pool{
		New: func() any {
			return image.NewRGBA(screenBounds)
		},
	}
	bounds := image.Rect(0, 0, p.Width, p.Height)
	s.downscaledImgBuffPool = sync.Pool{
		New: func() any {
			return image.NewRGBA(bounds)
		},
	}

	reader := s.reader

	r := video.ReaderFunc(func() (img image.Image, release func(), err error) {
		<-s.tick.C

		imgBuf := s.imgBuffPool.Get().(*image.RGBA)
		err = reader.Read().ToRGBA(imgBuf)
		if err != nil {
			s.imgBuffPool.Put(imgBuf)
			return nil, nil, err
		}

		if !isDiff {
			downscaledImgBuf := s.downscaledImgBuffPool.Get().(*image.RGBA)
			draw.NearestNeighbor.Scale(downscaledImgBuf, downscaledImgBuf.Rect, imgBuf,
				imgBuf.Bounds(), draw.Over, nil)
			img = downscaledImgBuf
			release = func() {
				s.imgBuffPool.Put(imgBuf)
				s.downscaledImgBuffPool.Put(downscaledImgBuf)
			}
			return
		}

		img = imgBuf
		release = func() {
			s.imgBuffPool.Put(imgBuf)
		}
		return
	})
	return r, nil
}

func (s *Screen) Properties() []prop.Media {
	rect := s.reader.img.Bounds()
	w := rect.Dx()
	h := rect.Dy()
	return []prop.Media{
		{
			DeviceID: deviceID(s.num),
			Video: prop.Video{
				Width:       w,
				Height:      h,
				FrameFormat: frame.FormatRGBA,
			},
		},
	}
}
