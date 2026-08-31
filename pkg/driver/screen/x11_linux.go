package screen

import (
	"fmt"
	"image"
	"time"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
	"golang.org/x/image/draw"
)

type Screen struct {
	num    int
	reader *reader
	tick   *time.Ticker
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
	s.tick = time.NewTicker(time.Duration(float32(time.Second) / p.FrameRate))
	var (
		dst           *image.RGBA
		downscaledImg = image.NewRGBA(image.Rect(0, 0, p.Width, p.Height))
	)
	reader := s.reader

	r := video.ReaderFunc(func() (image.Image, func(), error) {
		<-s.tick.C
		dst = reader.Read().ToRGBA(dst)
		draw.NearestNeighbor.Scale(downscaledImg, downscaledImg.Rect, dst,
			dst.Bounds(), draw.Over, nil)
		return downscaledImg, func() {}, nil
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
