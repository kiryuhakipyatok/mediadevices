//go:build !linux
// +build !linux

package screen

import (
	"fmt"
	"image"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	dgxi "github.com/ghp3000/screenshot"
	"github.com/kbinani/screenshot"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type Screen struct {
	displayIndex int
	doneCh       chan struct{}
	shot         dgxi.ScreenShot
	mu           sync.Mutex
	imgBuffPool  sync.Pool
}

func init() {
	Initialize()
}

// Initialize finds and registers active displays. This is part of an experimental API.
func Initialize() {
	activeDisplays := screenshot.NumActiveDisplays()
	for i := 0; i < activeDisplays; i++ {
		priority := driver.PriorityNormal
		if i == 0 {
			priority = driver.PriorityHigh
		}

		s := NewScreen(i)
		driver.GetManager().Register(s, driver.Info{
			Label:      fmt.Sprint(i),
			DeviceType: driver.Screen,
			Priority:   priority,
		})
	}
}

func NewScreen(displayIndex int) *Screen {
	s := Screen{
		displayIndex: displayIndex,
	}
	return &s
}

func (s *Screen) Open() error {
	s.doneCh = make(chan struct{})
	return nil
}

func (s *Screen) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.doneCh)
	if s.shot != nil {
		s.shot.Release()
		s.shot = nil
	}
	return nil
}

func (s *Screen) VideoRecord(selectedProp prop.Media) (video.Reader, string, error) {
	shot := dgxi.NewScreenShot(0)
	if err := shot.Init(s.displayIndex); err != nil {
		return nil, "", err
	}
	captureName := shot.GetCaptureName()
	bounds := shot.GetBounds()
	shot.DrawCursor(1)
	s.mu.Lock()
	s.shot = shot
	s.mu.Unlock()
	var j atomic.Int32
	s.imgBuffPool = sync.Pool{
		New: func() any {
			return image.NewRGBA(bounds)
		},
	}

	r := video.ReaderFunc(func() (img image.Image, release func(), err error) {
		runtime.LockOSThread()
		for {
			select {
			case <-s.doneCh:
				return nil, nil, io.EOF
			default:
			}

			s.mu.Lock()
			if s.shot == nil {
				s.mu.Unlock()
				return nil, nil, io.EOF
			}
			s.mu.Unlock()
			imgBuf := s.imgBuffPool.Get().(*image.RGBA)
			s.mu.Lock()
			err = s.shot.Capture(imgBuf)
			s.mu.Unlock()
			if err != nil {
				j.Add(1)
				fmt.Println("PUTTING FRAME BUFFER", j.Load())
				s.imgBuffPool.Put(imgBuf)
				if err.Error() == "no image yet" {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				return nil, nil, err
			}
			img = imgBuf
			release = func() {
				j.Add(1)
				fmt.Println("PUTTING FRAME BUFFER", j.Load())
				s.imgBuffPool.Put(imgBuf)
			}
			return
		}

	})
	return r, captureName, nil
}

func (s *Screen) Properties() []prop.Media {
	resolution := screenshot.GetDisplayBounds(s.displayIndex)
	supportedProp := prop.Media{
		Video: prop.Video{
			Width:       resolution.Dx(),
			Height:      resolution.Dy(),
			FrameFormat: frame.FormatRGBA,
		},
	}
	return []prop.Media{supportedProp}
}
