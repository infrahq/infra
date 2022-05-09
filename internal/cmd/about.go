package cmd

import (
	"math"
	"time"

	sprite "github.com/pdevine/go-asciisprite"
	tm "github.com/pdevine/go-asciisprite/termbox"
	"github.com/spf13/cobra"
)

type triangle struct {
	A     *point3D
	B     *point3D
	C     *point3D
	Color rune
}

func newAboutCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "about",
		Short:  "Display information about Infra",
		Args:   NoArgs,
		Group:  "Other commands:",
		Hidden: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return about()
		},
	}
}

func newTriangle(a, b, c *point3D, ch rune) *triangle {
	t := &triangle{
		A:     a,
		B:     b,
		C:     c,
		Color: ch,
	}
	return t
}

func (t *triangle) Draw(surf *sprite.Surface) {
	err := surf.Triangle(
		t.A.ScreenX(), t.A.ScreenY(),
		t.B.ScreenX(), t.B.ScreenY(),
		t.C.ScreenX(), t.C.ScreenY(),
		t.Color, true)
	//nolint
	if err != nil {
		// don't do anything
	}
}

type point3D struct {
	X     float64
	Y     float64
	Z     float64
	cX    float64
	cY    float64
	cZ    float64
	fl    float64
	vpX   float64
	vpY   float64
	scale float64
}

func newPoint3D(x, y, z float64) *point3D {
	p := &point3D{
		fl:    250.0,
		scale: 1.0,
		X:     x,
		Y:     y,
		Z:     z,
	}
	return p
}

func (p *point3D) SetVanishingPoint(vpX, vpY int) {
	p.vpX = float64(vpX)
	p.vpY = float64(vpY)
}

func (p *point3D) SetCenter(cX, cY, cZ float64) {
	p.cX = cX
	p.cY = cY
	p.cZ = cZ
}

func (p *point3D) ScreenX() int {
	p.scale = p.fl / (p.fl + p.Z + p.cZ)
	return int(math.Round(p.vpX + (p.cX+p.X)*p.scale))
}

func (p *point3D) ScreenY() int {
	p.scale = p.fl / (p.fl + p.Z + p.cZ)
	return int(math.Round(p.vpY + (p.cY+p.Y)*p.scale))
}

func (p *point3D) RotateX(angleX float64) {
	cosX := math.Cos(angleX)
	sinX := math.Sin(angleX)

	y := (p.Y * cosX) - (p.Z * sinX)
	z := (p.Z * cosX) + (p.Y * sinX)

	p.Y = y
	p.Z = z
}

func (p *point3D) RotateY(angleY float64) {
	cosY := math.Cos(angleY)
	sinY := math.Sin(angleY)

	x := (p.X * cosY) - (p.Z * sinY)
	z := (p.Z * cosY) + (p.X * sinY)

	p.X = x
	p.Z = z
}

func (p *point3D) RotateZ(angleZ float64) {
	cosZ := math.Cos(angleZ)
	sinZ := math.Sin(angleZ)

	x := (p.X * cosZ) - (p.Y * sinZ)
	y := (p.Y * cosZ) + (p.X * sinZ)

	p.X = x
	p.Y = y
}

type scrollText struct {
	sprite.BaseSprite
	Timer     int
	TimeOut   int
	finishing bool
	finished  bool
}

func newScrollText(width, height int) *scrollText {
	s := &scrollText{
		BaseSprite: sprite.BaseSprite{
			Visible: true,
		},
		TimeOut: 2,
	}
	f := sprite.NewJRSMFont()

	text := []string{
		"Contributors",
		"",
		"BruceMacD",
		"dnephin",
		"FSHA",
		"hoyyeva",
		"jmorganca",
		"kimskimchi",
		"mchiang0610",
		"mxyng",
		"pdevine",
		"ssoroka",
		"technovangelist",
		"",
		"",
		"Join us!",
		"Visit github.com/infrahq/infra",
	}

	buf := ""
	for _, t := range text {
		buf += f.BuildString(t)
	}

	surf := sprite.NewSurfaceFromString(buf, true)
	s.BlockCostumes = append(s.BlockCostumes, &surf)
	s.SetCostume(0)

	s.X = width/2 - surf.Width/2
	s.Y = height + 20

	return s
}

func (s *scrollText) Update() {
	s.Timer++
	if !s.finishing && s.Y+s.Height < 0 {
		s.TimeOut = 10
		s.Timer = 0
		s.finishing = true
	}

	if s.Timer >= s.TimeOut {
		s.Y -= 1
		s.Timer = 0
		if s.finishing {
			s.finished = true
		}
	}
}

type logo3D struct {
	sprite.BaseSprite
	points    []*point3D
	triangles []*triangle
	angleX    float64
	angleY    float64
	angleZ    float64
	width     int
	height    int
}

func newLogo3D(width, height int) *logo3D {
	s := &logo3D{
		BaseSprite: sprite.BaseSprite{
			Visible: true,
		},
		angleX: 0.05,
		angleY: 0.1,
		angleZ: 0.02,
		width:  width,
		height: height,
	}

	ps := []struct {
		X, Y, Z float64
	}{
		// i
		{-65, -10, 10},
		{-55, -10, 10},
		{-65, 20, 10},
		{-55, 20, 10},

		// n
		{-50, -10, 10},
		{-40, -10, 10},
		{-50, 20, 10},
		{-40, 20, 10},
		{-30, 0, 10},
		{-20, -5, 10},
		{-30, 20, 10},
		{-20, 20, 10},
		{-40, -7, 10},
		{-40, 0, 10},
		{-32, -2, 10},
		{-35, -10, 10},
		{-30, -10, 10},
		{-22, -7, 10},

		// f
		{-15, -10, 10},
		{5, -10, 10},
		{-15, 0, 10},
		{5, 0, 10},
		{-10, 0, 10},
		{0, 0, 10},
		{-10, 20, 10},
		{0, 20, 10},
		{-10, -10, 10},
		{0, -10, 10},
		{-10, -15, 10},
		{0, -12, 10},
		{2, -15, 10},
		{5, -15, 10},
		{5, -25, 10},
		{-3, -23, 10},
		{-7, -20, 10},

		// r
		{10, -10, 10},
		{20, -10, 10},
		{10, 20, 10},
		{20, 20, 10},
		{20, -7, 10},
		{23, -9, 10},
		{23, 1, 10},
		{20, 3, 10},
		{30, -10, 10},
		{30, 0, 10},

		// a
		{55, -10, 10},
		{65, -10, 10},
		{55, 20, 10},
		{65, 20, 10},
		{55, -7, 10},
		{55, 7, 10},
		{50, 0, 10},
		{53, -10, 10},
		{45, 1, 10},
		{40, -7, 10},
		{37, -1, 10},
		{43, 5, 10},
		{35, 7, 10},
		{45, 10, 10},
		{36, 15, 10},
		{40, 18, 10},
		{50, 10, 10},
		{50, 20, 10},
		{55, 7, 10},
		{55, 17, 10},
	}

	t := []struct {
		A, B, C int
	}{
		{0, 1, 3},
		{0, 3, 2},

		{4, 5, 7},
		{4, 7, 6},
		{8, 9, 10},
		{9, 10, 11},
		{12, 14, 13},
		{15, 8, 12},
		{16, 8, 15},
		{17, 8, 16},
		{9, 8, 17},

		{18, 19, 20},
		{19, 21, 20},
		{22, 23, 24},
		{23, 25, 24},
		{29, 27, 26},
		{28, 29, 26},
		{30, 29, 34},
		{34, 29, 28},
		{33, 30, 34},
		{32, 30, 33},
		{32, 31, 30},

		{35, 36, 37},
		{38, 37, 36},
		{39, 41, 42},
		{40, 41, 39},
		{43, 41, 40},
		{43, 44, 41},

		{45, 46, 47},
		{46, 48, 47},
		{49, 50, 51},
		{49, 51, 52},
		{51, 53, 52},
		{52, 53, 54},
		{54, 53, 55},
		{55, 53, 56},
		{55, 56, 57},
		{56, 58, 57},
		{57, 58, 59},
		{58, 61, 59},
		{59, 61, 60},
		{61, 62, 60},
		{61, 63, 62},
		{63, 64, 62},
	}

	for _, p := range ps {
		np := newPoint3D(p.X, p.Y, p.Z)
		np.SetVanishingPoint(s.width/2, s.height/2)
		np.SetCenter(0, 0, 100)
		s.points = append(s.points, np)
	}

	for _, p := range t {
		s.triangles = append(s.triangles, newTriangle(s.points[p.A], s.points[p.B], s.points[p.C], 'g'))
	}

	surf := sprite.NewSurface(s.width, s.height, false)
	s.BlockCostumes = append(s.BlockCostumes, &surf)

	return s
}

func (s *logo3D) Update() {
	for _, p := range s.points {
		p.RotateX(s.angleX)
		p.RotateY(s.angleY)
		p.RotateZ(s.angleZ)
	}

	surf := sprite.NewSurface(s.width, s.height, false)
	for _, t := range s.triangles {
		t.Draw(&surf)
	}

	s.BlockCostumes[0] = &surf
}

func about() error {
	err := tm.Init()
	if err != nil {
		panic(err)
	}
	defer tm.Close()

	w, h := tm.Size()

	eventQueue := make(chan tm.Event)
	go func() {
		for {
			eventQueue <- tm.PollEvent()
		}
	}()

	sprite.ColorMap['g'] = tm.ColorGray

	i := newLogo3D(w*2, h*2)
	st := newScrollText(w*2, h*2)

	var allSprites sprite.SpriteGroup
	allSprites.Init(w*2, h*2, true)
	allSprites.BlockMode = true
	allSprites.Background = tm.Attribute(1)
	allSprites.Sprites = append(allSprites.Sprites, i)
	allSprites.Sprites = append(allSprites.Sprites, st)

mainloop:
	for {
		tm.Clear(tm.Attribute(1), tm.Attribute(1))

		select {
		case ev := <-eventQueue:
			if ev.Type == tm.EventKey {
				switch {
				case ev.Key == tm.KeyEsc:
					break mainloop
				case ev.Key == tm.KeyArrowRight:
					i.angleX += 0.01
				case ev.Key == tm.KeyArrowLeft:
					i.angleX -= 0.01
				case ev.Key == tm.KeyArrowUp:
					i.angleY += 0.01
				case ev.Key == tm.KeyArrowDown:
					i.angleY -= 0.01
				case ev.Ch == 'a':
					i.angleZ += 0.01
				case ev.Ch == 'z':
					i.angleZ -= 0.01
				}
			} else if ev.Type == tm.EventResize {
				allSprites.Resize(w*2, h*2)
			}
		default:
			allSprites.Update()
			allSprites.Render()
			time.Sleep(50 * time.Millisecond)
			if st.finished {
				break mainloop
			}
		}
	}
	return nil
}
