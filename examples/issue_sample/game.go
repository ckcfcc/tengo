package issue_sample

import (
	"context"
	"fmt"
	"github.com/ckcfcc/tengo/v2"
	"github.com/ckcfcc/tengo/v2/stdlib"
	"image/color"
	"log"
)

package main

import (
"context"
"fmt"
"image/color"
"log"

"github.com/d5/tengo/v2"
"github.com/d5/tengo/v2/stdlib"

"github.com/hajimehoshi/ebiten/v2"
"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	tengoDoneWithFrameCh = make(chan int)
	tengoStartFrameCh    = make(chan int)
	frameNum             = 0
	playerModuleCode     = []byte(`
engine := import("engine")
rand := import("rand")
newVec2 := import("vec2")

move := func(self, spriteIdx) {
	return func() {
		self.pos.add(self.delta)
		if self.pos.x < 0 || self.pos.x > 320 {
			self.delta.x=-self.delta.x
		}
		if self.pos.y < 0 || self.pos.y > 240 {
			self.delta.y=-self.delta.y
		}
		engine.sprites[spriteIdx].move(self.pos.x, self.pos.y)
	}
}

export func(spriteIdx, x,y) {
	self := {
		pos: newVec2(x,y),
		delta: newVec2(5,5)
	}
	self.move = move(self, spriteIdx)
	return self
}
`)
	vec2ModuleCode = []byte(`
add := func(self) {
	return func(otherVec2) {
		self.x += otherVec2.x
		self.y += otherVec2.y
	}
}

export func(x,y) {
	self := {
		x: float(x),
		y: float(y)
	}
	self.add = add(self)
	return self
}
`)
	script = []byte(
		`
fmt := import("fmt")
rand := import("rand")
engine := import("engine")

newPlayer := import("player")

p1 := newPlayer(0, rand.float()*320, rand.float()*240)
p2 := newPlayer(1, rand.float()*320, rand.float()*240)

for {
	p1.move()
	p2.move()
	fmt.println("frame", " ", p1, " ", p2)
	engine.WaitForNextFrame()
}

`)
)

//// Sprite

type Sprite struct {
	X, Y     float64
	Image    *ebiten.Image
	DrawOpts ebiten.DrawImageOptions
}

func NewSprite() *Sprite {
	s := &Sprite{}
	s.setup()
	return s
}

func (s *Sprite) setup() *Sprite {
	s.Image = ebiten.NewImage(32, 32)
	s.Image.Fill(color.White)
	return s
}

func (s *Sprite) Draw(screen *ebiten.Image) {
	s.DrawOpts.GeoM.Reset()
	s.DrawOpts.GeoM.Translate(-16, -16)
	s.DrawOpts.GeoM.Translate(s.X, s.Y)
	screen.DrawImage(s.Image, &s.DrawOpts)
}

func (s *Sprite) GetTengoObject() tengo.Object {
	return &tengo.Map{
		Value: map[string]tengo.Object{
			"move": &tengo.UserFunction{Name: "move", Value: s.move},
		},
	}
}

func (s *Sprite) move(args ...tengo.Object) (ret tengo.Object, err error) {
	if len(args) != 2 {
		return nil, tengo.ErrWrongNumArguments
	}

	var ok bool

	if s.X, ok = tengo.ToFloat64(args[0]); !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "x", Expected: "float64", Found: args[0].TypeName()}
	}

	if s.Y, ok = tengo.ToFloat64(args[1]); !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "y", Expected: "float64", Found: args[0].TypeName()}
	}

	return tengo.UndefinedValue, nil
}

//// Game

type Game struct {
	Sprites       []*Sprite
	contextCancel func()
}

func (g *Game) Update() error {
	tengoStartFrameCh <- frameNum // tell tengo to start a new frame
	<-tengoDoneWithFrameCh        // wait for tengo to finish that frame
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, sprite := range g.Sprites {
		sprite.Draw(screen)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("[ %d ]", frameNum))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

func (g *Game) waitForNextFrame(args ...tengo.Object) (ret tengo.Object, err error) {
	if frameNum > 0 { // otherwise deadlock. someone has to go first.
		tengoDoneWithFrameCh <- frameNum // inform outer that we're done with the previous frame
	}
	frameNum++
	fmt.Println(frameNum)
	if frameNum == 2000 {
		g.contextCancel()
	}
	<-tengoStartFrameCh // wait for the new frame before letting tengo continue
	return tengo.UndefinedValue, nil
}

//// main

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	game := &Game{
		Sprites: []*Sprite{
			NewSprite(),
			NewSprite(),
		},
		contextCancel: cancel,
	}

	engineModule := &tengo.BuiltinModule{
		Attrs: map[string]tengo.Object{
			"WaitForNextFrame": &tengo.UserFunction{
				Name:  "WaitForNextFrame",
				Value: game.waitForNextFrame,
			},
			"sprites": &tengo.Array{
				Value: []tengo.Object{
					game.Sprites[0].GetTengoObject(),
					game.Sprites[1].GetTengoObject(),
				},
			},
		},
	}

	script := tengo.NewScript(script)

	moduleMap := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	moduleMap.AddSourceModule("player", playerModuleCode)
	moduleMap.AddSourceModule("vec2", vec2ModuleCode)
	moduleMap.Add("engine", engineModule)
	script.SetImports(moduleMap)

	go func() { // run the script in its own goroutine
		compiled, err := script.RunContext(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(compiled.GetAll())
	}()

	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Hello, World!")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
