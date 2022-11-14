package ino

import (
	"flag"
	"fmt"
	_ "image/png"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/text/language"

	"github.com/hajimehoshi/go-inovation/ino/internal/audio"
	"github.com/hajimehoshi/go-inovation/ino/internal/draw"
	"github.com/hajimehoshi/go-inovation/ino/internal/input"
	"github.com/hajimehoshi/go-inovation/ino/internal/lang"
)

type Game struct {
	resourceLoadedCh chan error
	scene            Scene
	gameData         *GameData
	lang             language.Tag
	cpup             *os.File
	transparent      bool
}

var (
	cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	mute       = flag.Bool("mute", false, "mute")
)

func (g *Game) SetTransparent() {
	g.transparent = true
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) Update() error {
	if g.resourceLoadedCh != nil {
		select {
		case err := <-g.resourceLoadedCh:
			if err != nil {
				return err
			}
			g.resourceLoadedCh = nil
		default:
			return nil
		}
	}

	input.Current().Update()

	if input.Current().IsKeyJustPressed(ebiten.KeyF) {
		f := ebiten.IsFullscreen()
		ebiten.SetFullscreen(!f)
		if f {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		}
	}

	if input.Current().IsKeyJustPressed(ebiten.KeyP) && *cpuProfile != "" && g.cpup == nil {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			panic(err)
		}
		g.cpup = f
		pprof.StartCPUProfile(f)
		fmt.Println("Start CPU Profiling")
	}

	if input.Current().IsKeyJustPressed(ebiten.KeyQ) && g.cpup != nil {
		pprof.StopCPUProfile()
		g.cpup.Close()
		g.cpup = nil
		fmt.Println("Stop CPU Profiling")
	}

	if g.scene == nil {
		g.scene = &TitleScene{}
	} else {
		switch g.scene.Msg() {
		case GAMESTATE_MSG_REQ_TITLE:
			audio.PauseBGM()
			g.scene = &TitleScene{}
		case GAMESTATE_MSG_REQ_OPENING:
			if err := audio.PlayBGM(audio.BGM1); err != nil {
				return err
			}
			g.scene = &OpeningScene{}
		case GAMESTATE_MSG_REQ_GAME:
			g.scene = NewGameScene(g)
		case GAMESTATE_MSG_REQ_ENDING:
			if err := audio.PlayBGM(audio.BGM1); err != nil {
				return err
			}
			g.scene = &EndingScene{}
		case GAMESTATE_MSG_REQ_SECRET_COMMAND:
			if err := audio.PlayBGM(audio.BGM1); err != nil {
				return err
			}
			g.scene = NewSecretScene(SecretTypeCommand)
		case GAMESTATE_MSG_REQ_SECRET_CLEAR:
			if err := audio.PlayBGM(audio.BGM1); err != nil {
				return err
			}
			g.scene = NewSecretScene(SecretTypeClear)
		}
	}
	g.scene.Update(g)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.resourceLoadedCh != nil {
		ebitenutil.DebugPrint(screen, "Now Loading...")
		return
	}
	g.scene.Draw(screen, g)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("\nFPS: %.2f", ebiten.CurrentFPS()))
}

func NewGame() (*Game, error) {
	if *mute {
		audio.Mute()
	}

	game := &Game{
		resourceLoadedCh: make(chan error),
		lang:             lang.SystemLang(),
	}
	go func() {
		if err := draw.LoadImages(); err != nil {
			game.resourceLoadedCh <- err
			return
		}
		if err := audio.Load(); err != nil {
			game.resourceLoadedCh <- err
			return
		}
		close(game.resourceLoadedCh)
	}()
	if err := audio.Finalize(); err != nil {
		return nil, err
	}

	if err := setIcons(); err != nil {
		return nil, err
	}

	return game, nil
}
