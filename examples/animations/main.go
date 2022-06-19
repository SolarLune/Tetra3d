package main

import (
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	examples.ExampleGame

	Library *tetra3d.Library

	Time float64

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugNormals   bool
	DrawDebugCenters   bool
	DrawDebugWireframe bool
}

//go:embed animations.gltf
var gltf []byte

func NewGame() *Game {

	game := &Game{
		ExampleGame:   examples.NewExampleGame(796, 448),
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(gltf, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	g.Scene = g.Library.Scenes[0]
	// Turn off lighting
	g.Scene.LightingOn = false

	g.SetupCameraAt(vector.Vector{0, 0, 0})
	g.Camera.Move(0, 0, 10)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	// newCube := scenes.Scenes[0].Root.Get("Cube.001").Clone()
	// scenes.Scenes[0].Root.Get("Armature/Root/1/2/3/4/5").AddChildren(newCube)
	// newCube.SetLocalPosition(vector.Vector{0, 2, 0})

}

func (g *Game) Update() error {
	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	g.ProcessCameraInputs()

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, g.Camera.ColorTexture())
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.StartProfiling()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.DrawDebugNormals = !g.DrawDebugNormals
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.DrawDebugCenters = !g.DrawDebugCenters
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	scene := g.Library.Scenes[0]

	armature := scene.Root.ChildrenRecursive().ByName("Armature", true)[0].(*tetra3d.Node)
	armature.Rotate(0, 1, 0, 0.01)

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		// armature.AnimationPlayer.FinishMode = tetra3d.FinishModeStop
		armature.AnimationPlayer().Play(g.Library.Animations["ArmatureAction"])
	}

	armature.AnimationPlayer().Update(1.0 / 60)

	table := scene.Root.Get("Table").(*tetra3d.Model)
	table.AnimationPlayer().BlendTime = 0.1
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		table.AnimationPlayer().Play(g.Library.Animations["SmoothRoll"])
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		table.AnimationPlayer().Play(g.Library.Animations["StepRoll"])
	}

	table.AnimationPlayer().Update(1.0 / 60)

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt := &ebiten.DrawImageOptions{}
	w, h := g.Camera.ColorTexture().Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.Red())
	}

	if g.DrawDebugNormals {
		g.Camera.DrawDebugNormals(screen, g.Scene.Root, 0.5, colors.Blue())
	}

	if g.DrawDebugCenters {
		g.Camera.DrawDebugCenters(screen, g.Scene.Root, colors.SkyBlue())
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n1 Key: Play [SmoothRoll] Animation On Table\n2 Key: Play [StepRoll] Animation on Table\nNote the animations can blend\nF Key: Play Animation on Skinned Mesh\nNote that the nodes move as well\nF4: Toggle fullscreen\nF6: Node Debug View\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 140, color.RGBA{255, 0, 0, 255})
	}

}

func (g *Game) StartProfiling() {
	outFile, err := os.Create("./cpu.pprof")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Beginning CPU profiling...")
	pprof.StartCPUProfile(outFile)
	go func() {
		time.Sleep(2 * time.Second)
		pprof.StopCPUProfile()
		fmt.Println("CPU profiling finished.")
	}()
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Animations Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
