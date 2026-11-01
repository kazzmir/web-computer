package pygame

import (
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/kazzmir/web-computer/gpython/py"
)

func decodeWAV(name string) ([]byte, error) {
	file, err := openAsset(name)
	if err != nil {
		return nil, py.ExceptionNewf(py.OSError, "%v", err)
	}
	defer file.Close()
	stream, err := wav.DecodeWithSampleRate(44100, file)
	if err != nil {
		return nil, py.ExceptionNewf(py.OSError, "%v", err)
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, py.ExceptionNewf(py.OSError, "%v", err)
	}
	return data, nil
}

// repeats is the number of additional plays, or -1 for an infinite loop.
// Reading never duplicates the entire song in memory.
type repeatingPCM struct {
	data              []byte
	position, repeats int
	finished          bool
}

func (r *repeatingPCM) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if r.finished || len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(dst, r.data[r.position:])
	r.position += n
	if r.position == len(r.data) {
		r.position = 0
		if r.repeats == 0 {
			r.finished = true
		} else if r.repeats > 0 {
			r.repeats--
		}
	}
	return n, nil
}

type musicState struct {
	mu         sync.Mutex
	data       []byte
	player     *audio.Player
	fadeCancel chan struct{}
	fades      sync.WaitGroup
}

func (m *musicState) stop() {
	m.mu.Lock()
	if m.fadeCancel != nil {
		close(m.fadeCancel)
		m.fadeCancel = nil
	}
	if m.player != nil {
		m.player.Close()
		m.player = nil
	}
	m.mu.Unlock()
	m.fades.Wait()
}
func (m *musicState) fade(duration time.Duration) {
	if duration <= 0 {
		m.stop()
		return
	}
	m.mu.Lock()
	if m.fadeCancel != nil {
		close(m.fadeCancel)
		m.fadeCancel = nil
	}
	player := m.player
	if player == nil {
		m.mu.Unlock()
		return
	}
	cancel := make(chan struct{})
	m.fadeCancel = cancel
	volume := player.Volume()
	m.fades.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.fades.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		for {
			select {
			case <-cancel:
				return
			case <-ticker.C:
				m.mu.Lock()
				if m.player != player || m.fadeCancel != cancel {
					m.mu.Unlock()
					return
				}
				elapsed := time.Since(start)
				if elapsed >= duration {
					player.Close()
					m.player = nil
					m.fadeCancel = nil
					m.mu.Unlock()
					return
				}
				player.SetVolume(volume * (1 - float64(elapsed)/float64(duration)))
				m.mu.Unlock()
			}
		}
	}()
}

func musicMethods() ([]*py.Method, func()) {
	music := &musicState{}
	return []*py.Method{
		py.MustNewMethod("_music_load", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "s:load", []string{"filename"}, &obj); err != nil {
				return nil, err
			}
			data, err := decodeWAV(string(obj.(py.String)))
			if err != nil {
				return nil, err
			}
			music.stop()
			music.mu.Lock()
			music.data = data
			music.mu.Unlock()
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_music_play", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "i:play", []string{"loops"}, &obj); err != nil {
				return nil, err
			}
			if !audioAvailable() {
				return nil, py.ExceptionNewf(py.OSError, "audio device unavailable")
			}
			loops := int(obj.(py.Int))
			if loops < -1 {
				return nil, py.ExceptionNewf(py.ValueError, "loops must be -1 or nonnegative")
			}
			music.stop()
			music.mu.Lock()
			defer music.mu.Unlock()
			if music.data == nil {
				return nil, py.ExceptionNewf(py.OSError, "no music loaded")
			}
			player, err := getAudioContext().NewPlayer(&repeatingPCM{data: music.data, repeats: loops})
			if err != nil {
				return nil, py.ExceptionNewf(py.OSError, "%v", err)
			}
			music.player = player
			player.Play()
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_music_stop", func(self py.Object, args py.Tuple) (py.Object, error) {
			if err := py.UnpackTuple(args, nil, "stop", 0, 0); err != nil {
				return nil, err
			}
			music.stop()
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_music_fadeout", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "i:fadeout", []string{"milliseconds"}, &obj); err != nil {
				return nil, err
			}
			ms := obj.(py.Int)
			if ms < 0 {
				return nil, py.ExceptionNewf(py.ValueError, "fadeout duration must be nonnegative")
			}
			music.fade(time.Duration(ms) * time.Millisecond)
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_music_busy", func(self py.Object, args py.Tuple) (py.Object, error) {
			if err := py.UnpackTuple(args, nil, "get_busy", 0, 0); err != nil {
				return nil, err
			}
			music.mu.Lock()
			defer music.mu.Unlock()
			return py.Bool(music.player != nil && music.player.IsPlaying()), nil
		}, 0, ""),
	}, func() {
		music.stop()
		music.mu.Lock()
		music.data = nil
		music.mu.Unlock()
	}
}
