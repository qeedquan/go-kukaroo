package main

import (
	"flag"
	"fmt"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/qeedquan/go-media/sdl"
	"github.com/qeedquan/go-media/sdl/sdlgfx"
	"github.com/qeedquan/go-media/sdl/sdlimage"
	"github.com/qeedquan/go-media/sdl/sdlimage/sdlcolor"
	"github.com/qeedquan/go-media/sdl/sdlmixer"
)

var (
	conf struct {
		assets     string
		pref       string
		fullscreen bool
		music      bool
		sound      bool
		invincible bool
	}

	screen *Display
	fps    sdlgfx.FPSManager

	bg    Background
	intro Background

	song *sdlmixer.Music
	flap *sdlmixer.Chunk

	state func()
	level int
	won   bool

	walls    []Barrier
	enemies  []Enemy
	borders  []*Border
	feathers []*DroppedFeather
	player   *Player
)

func main() {
	runtime.LockOSThread()
	parseFlags()
	initSDL()
	loadAssets()
	playSong()

	rand.Seed(time.Now().UnixNano())
	for {
		screen.SetDrawColor(sdlcolor.Black)
		screen.Clear()
		state()
		screen.Present()
		fps.Delay()
	}
}

func ck(err error) {
	if err != nil {
		sdl.LogCritical(sdl.LOG_CATEGORY_APPLICATION, "%v", err)
		if screen != nil {
			sdl.ShowSimpleMessageBox(sdl.MESSAGEBOX_ERROR, "Error", err.Error(), screen.Window)
		}
		os.Exit(1)
	}
}

func ek(err error) bool {
	if err != nil {
		sdl.LogError(sdl.LOG_CATEGORY_APPLICATION, "%v", err)
		return true
	}
	return false
}

type Display struct {
	*sdl.Window
	*sdl.Renderer
}

func newDisplay(w, h int, flags sdl.WindowFlags) *Display {
	window, renderer, err := sdl.CreateWindowAndRenderer(w, h, flags)
	ck(err)
	return &Display{window, renderer}
}

func parseFlags() {
	conf.assets = filepath.Join(sdl.GetBasePath(), "assets")
	conf.pref = sdl.GetPrefPath("", "kukaroo")
	flag.StringVar(&conf.assets, "assets", conf.assets, "assets directory")
	flag.StringVar(&conf.pref, "pref", conf.pref, "preference directory")
	flag.BoolVar(&conf.sound, "sound", true, "enable sound")
	flag.BoolVar(&conf.music, "music", true, "enable music")
	flag.BoolVar(&conf.fullscreen, "fullscreen", false, "fullscreen mode")
	flag.BoolVar(&conf.invincible, "invincible", false, "be invincible")
	flag.Parse()

}

func initSDL() {
	err := sdl.Init(sdl.INIT_EVERYTHING &^ sdl.INIT_AUDIO)
	ck(err)

	err = sdl.InitSubSystem(sdl.INIT_AUDIO)
	ek(err)

	err = sdlmixer.OpenAudio(44100, sdl.AUDIO_S16, 2, 8192)
	ek(err)

	_, err = sdlmixer.Init(sdlmixer.INIT_OGG)
	ek(err)

	wflag := sdl.WINDOW_RESIZABLE
	if conf.fullscreen {
		wflag |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	screen = newDisplay(660, 390, wflag)

	screen.SetTitle("Kukaroo!")
	screen.SetLogicalSize(660, 390)
	screen.SetDrawColor(sdlcolor.Black)
	screen.Clear()
	screen.Present()

	sdl.ShowCursor(0)

	fps.Init()
	fps.SetRate(60)
}

func loadAssets() {
	song = loadMusic("Music.ogg")
	flap = loadSound("Flap.wav")

	bg = Background{-50, -50, loadTexture("BG.png")}
	intro = Background{0, 0, loadTexture("Intro.png")}

	state = outGame
	reset()
}

func reset() {
	level = 1
	loadLevel()
	player = newPlayer(50, 50)
	won = false
}

func loadMusic(name string) *sdlmixer.Music {
	name = filepath.Join(conf.assets, name)
	music, err := sdlmixer.LoadMUS(name)
	if ek(err) {
		return nil
	}
	return music
}

func loadSound(name string) *sdlmixer.Chunk {
	name = filepath.Join(conf.assets, name)
	chunk, err := sdlmixer.LoadWAV(name)
	if ek(err) {
		return nil
	}
	return chunk
}

var (
	textures = make(map[string]*sdl.Texture)
)

func loadTexture(name string) *sdl.Texture {
	name = filepath.Join(conf.assets, name)
	if texture, found := textures[name]; found {
		return texture
	}

	texture, err := sdlimage.LoadTextureFile(screen.Renderer, name)
	ck(err)
	textures[name] = texture
	return texture
}

func loadLevel() {
	name := filepath.Join(conf.assets, fmt.Sprintf("Level%d.png", level))
	f, err := os.Open(name)
	ck(err)
	defer f.Close()

	m, _, err := image.Decode(f)
	ck(err)

	enemies = enemies[:0]
	walls = walls[:0]
	borders = borders[:0]
	feathers = feathers[:0]

	r := m.Bounds()
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			cr, cg, cb, ca := m.At(x, y).RGBA()
			c := sdl.Color{uint8(cr), uint8(cg), uint8(cb), uint8(ca)}
			fx, fy := float64(x)*30-30, float64(y)*30-30
			switch c {
			case sdl.Color{255, 255, 0, 255}:
				enemies = append(enemies, newSawBlade(fx, fy))
			case sdl.Color{0, 0, 255, 255}:
				enemies = append(enemies, newElectricBox(fx, fy))
			case sdl.Color{0, 255, 255, 255}:
				enemies = append(enemies, newFallingBlock(fx, fy))
			case sdl.Color{255, 255, 255, 255}:
				walls = append(walls, newWall(fx, fy))
			case sdl.Color{255, 0, 255, 255}:
				walls = append(walls, newButton(fx, fy))
			case sdl.Color{255, 0, 0, 255}:
				borders = append(borders, newBorder(fx, fy, -1))
			case sdl.Color{0, 255, 0, 255}:
				borders = append(borders, newBorder(fx, fy, 1))
			}
		}
	}

	// this is a hack to make it so that if the player
	// is flying when he goes to the next level and it
	// hits a wall, it won't trap the player into a state
	// where he can't continue
	if player != nil {
		for _, b := range borders {
			if b.Rect().Collide(player.rect) {
				return
			}
		}

		if player.y < 0 {
			player.y = 28
		}
		for _, w := range walls {
			r := w.Rect()
			if r.Collide(player.rect) {
				if player.y >= float64(r.Y) {
					player.y = float64(r.Y + r.H)
					player.UpdateRect()
				}
			}
		}
	}
}

func playSong() {
	if song == nil || !conf.music {
		return
	}

	song.Play(-1)
}

func playFlap() {
	if flap == nil || !conf.sound {
		return
	}

	flap.PlayChannel(-1, 0)
}

func outGame() {
	intro.Blit()
	for {
		ev := sdl.PollEvent()
		if ev == nil {
			break
		}
		switch ev := ev.(type) {
		case sdl.QuitEvent:
			os.Exit(0)
		case sdl.KeyDownEvent:
			switch ev.Sym {
			case sdl.K_ESCAPE:
				os.Exit(0)
			case sdl.K_SPACE:
				if won {
					reset()
					intro = Background{0, 0, loadTexture("Intro.png")}
				}
				state = inGame
			}
		}
	}
}

func inGame() {
	for {
		ev := sdl.PollEvent()
		if ev == nil {
			break
		}
		switch ev := ev.(type) {
		case sdl.QuitEvent:
			os.Exit(0)
		case sdl.KeyDownEvent:
			switch ev.Sym {
			case sdl.K_ESCAPE:
				os.Exit(0)
			case sdl.K_a, sdl.K_LEFT:
				player.dx = -2
			case sdl.K_d, sdl.K_RIGHT:
				player.dx = 2
			case sdl.K_n:
				loadLevel()
			case sdl.K_p:
				state = outGame
			case sdl.K_SPACE, sdl.K_UP:
				playFlap()
				player.dy = -2.5
				n := rand.Intn(2) + 2
				for i := 0; i < n; i++ {
					x := player.x + float64(rand.Intn(11)-5)
					y := player.y + float64(rand.Intn(8))
					feathers = append(feathers, newDroppedFeather(x, y))
				}
			case sdl.K_BACKSPACE:
				conf.invincible = !conf.invincible
				sdl.Log("Invincibility toggle: %v", conf.invincible)
			}
		case sdl.KeyUpEvent:
			switch ev.Sym {
			case sdl.K_d, sdl.K_a, sdl.K_LEFT, sdl.K_RIGHT:
				player.dx *= .5
			}
		}
	}
	bg.Blit()
	for _, w := range walls {
		w.Blit()
	}
	for i := 0; i < len(feathers); {
		f := feathers[i]
		f.Fall()
		f.Blit()
		if f.t <= 0 {
			l := len(feathers) - 1
			feathers[i], feathers = feathers[l], feathers[:l]
		} else {
			i++
		}
	}
	for _, e := range enemies {
		e.Move()
		e.Blit()
		r := e.Rect()
		if r.Collide(player.rect) && !conf.invincible {
			player.ReturnToLast()
			player.dy = 0
			loadLevel()
		}
	}
	player.Move()
	player.Blit()

	for _, b := range borders {
		r := b.Rect()
		if r.Collide(player.rect) {
			level += b.level
			if player.x > 630 {
				player.x = 5
				bg.x -= 20
			} else if player.x < 10 {
				player.x = 610
				bg.x += 20
			} else if player.y > 350 {
				player.y = 5
				bg.y -= 20
			} else if player.y < 10 {
				player.y = 360
				bg.y += 20
			}

			player.lx, player.ly = player.x, player.y
			player.UpdateRect()
			loadLevel()
		}
	}

	if level > 20 {
		intro = Background{0, 0, loadTexture("Finish.jpg")}
		won = true
		state = outGame
	}
}

type Background struct {
	x, y    int
	texture *sdl.Texture
}

func (b *Background) Blit() {
	_, _, w, h, err := b.texture.Query()
	ck(err)
	screen.Copy(b.texture, nil, &sdl.Rect{int32(b.x), int32(b.y), int32(w), int32(h)})
}

type Block struct {
	x, y, dx, dy float64
	rect         sdl.Rect
	degrees      float64
	texture      *sdl.Texture
	sheet        [2]*sdl.Texture
	num          int
}

func newBlock(x, y float64) Block {
	return Block{
		x:    x,
		y:    y,
		rect: sdl.Rect{int32(x), int32(y), 28, 28},
	}
}

func (b Block) Blit() {
	_, _, w, h, err := b.texture.Query()
	ck(err)
	r := &sdl.Rect{int32(b.x), int32(b.y), int32(w), int32(h)}
	screen.CopyEx(b.texture, nil, r, -b.degrees, nil, sdl.FLIP_NONE)
}

func (b *Block) UpdateRect() {
	b.rect.X, b.rect.Y = int32(b.x), int32(b.y)
}

func (b Block) Rect() sdl.Rect {
	return b.rect
}

type Button struct {
	Block
}

func newButton(x, y float64) Button {
	b := Button{
		Block: newBlock(x, y),
	}
	b.texture = loadTexture("Button.png")
	return b
}

type Barrier interface {
	Rect() sdl.Rect
	Blit()
}

type Wall struct {
	Block
}

func newWall(x, y float64) *Wall {
	w := &Wall{
		Block: newBlock(x, y),
	}
	w.texture = loadTexture("Crate.png")
	return w
}

type Border struct {
	Block
	level int
}

func newBorder(x, y float64, level int) *Border {
	b := &Border{
		Block: newBlock(x, y),
	}
	b.level = level
	return b
}

type SawBlade struct {
	Block
}

func newSawBlade(x, y float64) *SawBlade {
	s := &SawBlade{
		Block: newBlock(x, y),
	}
	s.sheet[0] = loadTexture("Sawblade0.png")
	s.sheet[1] = loadTexture("Sawblade1.png")
	s.texture = s.sheet[0]
	s.dx = rand.Float64()*4 - 2
	return s
}

func (s *SawBlade) Move() {
	s.x += s.dx
	s.texture = s.sheet[s.num]
	s.num = (s.num + 1) % len(s.sheet)
	s.UpdateRect()

	for _, w := range walls {
		if w.Rect().Collide(s.rect) {
			s.dx = -s.dx
			s.x += s.dx
		}
	}
}

type ElectricBox struct {
	Block
}

func newElectricBox(x, y float64) *ElectricBox {
	e := &ElectricBox{
		Block: newBlock(x, y),
	}
	e.sheet[0] = loadTexture("Electric0.png")
	e.sheet[1] = loadTexture("Electric1.png")
	e.texture = e.sheet[0]
	return e
}

func (e *ElectricBox) Move() {
	if e.num >= 4 {
		e.texture = e.sheet[0]
	} else if e.num < 4 {
		e.texture = e.sheet[1]
	}
	if e.num++; e.num > 8 {
		e.num = 0
	}
}

type FallingBlock struct {
	Block
}

func newFallingBlock(x, y float64) *FallingBlock {
	f := &FallingBlock{
		Block: newBlock(x, y),
	}
	f.texture = loadTexture("Crate.png")
	return f
}

func (f *FallingBlock) Move() {
	f.y += f.dy
	f.UpdateRect()
	if f.x-player.x < 40 {
		f.dy = 9
	}
}

type Enemy interface {
	Move()
	Blit()
	Rect() sdl.Rect
}

type Player struct {
	Block
	lx, ly float64
	flip   sdl.RendererFlip
}

func newPlayer(x, y float64) *Player {
	p := &Player{
		Block: newBlock(x, y),
	}
	p.sheet[0] = loadTexture("Canary0.png")
	p.sheet[1] = loadTexture("Canary1.png")
	p.texture = p.sheet[0]
	p.lx, p.ly = p.x, p.y
	return p
}

func (p *Player) Move() {
	p.x += p.dx
	p.UpdateRect()
	for _, w := range walls {
		r := w.Rect()
		if r.Collide(p.rect) {
			if p.dx > 0 {
				p.rect.X = r.X - p.rect.W
			} else if p.dx < 0 {
				p.rect.X = r.X + r.W
			}
			p.x = float64(p.rect.X)
			p.UpdateRect()
		}
	}
	p.gravity()
}

func (p *Player) gravity() {
	p.dy += 0.08
	p.y += p.dy
	p.UpdateRect()
	if p.dy < 0 {
		p.degrees = 0
		p.texture = p.sheet[0]
		if p.dx > 0 {
			p.flip = sdl.FLIP_NONE
		} else {
			p.flip = sdl.FLIP_HORIZONTAL
		}

		for _, w := range walls {
			r := w.Rect()
			if r.Collide(p.rect) {
				p.dy = 0
				p.rect.Y = r.Y + r.H
				p.y = float64(p.rect.Y)
				p.UpdateRect()
			}
		}

	} else if p.dy >= 0 {
		p.texture = p.sheet[1]
		if p.dx > 0 {
			p.degrees -= .7
			p.flip = sdl.FLIP_NONE
		} else {
			p.degrees += .7
			p.flip = sdl.FLIP_HORIZONTAL
		}

		for _, w := range walls {
			r := w.Rect()
			if r.Collide(p.rect) {
				p.rect.Y = r.Y - p.rect.H
				p.y = float64(p.rect.Y)
				p.UpdateRect()
				p.dy = 0
				p.degrees = 0
			}
		}
	}
}

func (p *Player) ReturnToLast() {
	p.x, p.y = p.lx, p.ly
	p.UpdateRect()
}

func (p *Player) Blit() {
	_, _, w, h, err := p.texture.Query()
	ck(err)
	rect := sdl.Rect{int32(p.x) - 5, int32(p.y), int32(w), int32(h)}
	if p.degrees != 0 {
		rect.Y -= 2
	}

	screen.CopyEx(p.texture, nil, &rect, -p.degrees, nil, p.flip)
}

type DroppedFeather struct {
	Block
	ox, oy  float64
	dir     int
	maxDist int
	t       int
}

func newDroppedFeather(x, y float64) *DroppedFeather {
	d := &DroppedFeather{
		Block: newBlock(x, y),
	}
	d.ox, d.oy = d.x, d.y
	d.dx = (2 + rand.Float64()*2) * .3
	d.dy = (2 + rand.Float64()*3) * .2
	if rand.Float64() >= .5 {
		d.dir = 1
	} else {
		d.dir = -1
	}
	d.maxDist = rand.Intn(56) - 10
	d.texture = loadTexture("Feather.png")
	d.t = 250
	return d
}

func (d *DroppedFeather) Fall() {
	switch d.dir {
	case -1:
		d.x -= d.dx
		d.degrees += .5
		if d.dx < d.ox-float64(d.maxDist) {
			d.dx = 1
		}
	case 1:
		d.x += d.dx
		d.degrees -= .5
		if d.x > d.ox-float64(d.maxDist) {
			d.dx = -1
		}
	}
	d.y += d.dy
	d.UpdateRect()
	d.t--
}
