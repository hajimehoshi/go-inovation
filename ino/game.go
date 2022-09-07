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

	"strconv"
	"strings"
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
	return ScreenWidth*2, ScreenHeight*2
}

func (g *Game) Update() error {
	return nil

	if g.resourceLoadedCh != nil {
		select {
		case err := <-g.resourceLoadedCh:
			if err != nil {
				return err
			}
			g.resourceLoadedCh = nil
		default:
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
	ids := ebiten.AppendGamepadIDs(nil)
	if len(ids) > 0 {
		ebitenutil.DebugPrint(screen, gamepadTest(ids[0]))
	} else {
		ebitenutil.DebugPrint(screen, "no gamepad")
	}
	return

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

var standardButtonToString = map[ebiten.StandardGamepadButton]string{
	ebiten.StandardGamepadButtonRightBottom:      "RB",
	ebiten.StandardGamepadButtonRightRight:       "RR",
	ebiten.StandardGamepadButtonRightLeft:        "RL",
	ebiten.StandardGamepadButtonRightTop:         "RT",
	ebiten.StandardGamepadButtonFrontTopLeft:     "FTL",
	ebiten.StandardGamepadButtonFrontTopRight:    "FTR",
	ebiten.StandardGamepadButtonFrontBottomLeft:  "FBL",
	ebiten.StandardGamepadButtonFrontBottomRight: "FBR",
	ebiten.StandardGamepadButtonCenterLeft:       "CL",
	ebiten.StandardGamepadButtonCenterRight:      "CR",
	ebiten.StandardGamepadButtonLeftStick:        "LS",
	ebiten.StandardGamepadButtonRightStick:       "RS",
	ebiten.StandardGamepadButtonLeftBottom:       "LB",
	ebiten.StandardGamepadButtonLeftRight:        "LR",
	ebiten.StandardGamepadButtonLeftLeft:         "LL",
	ebiten.StandardGamepadButtonLeftTop:          "LT",
	ebiten.StandardGamepadButtonCenterCenter:     "CC",
}

func gamepadTest(id ebiten.GamepadID) string {
	var str string
	var standard string
	if ebiten.IsStandardGamepadLayoutAvailable(id) {
		standard = " (Standard Layout)"
	}

	maxAxis := ebiten.GamepadAxisCount(id)
	var axes []string
	for a := 0; a < maxAxis; a++ {
		v := ebiten.GamepadAxisValue(id, a)
		axes = append(axes, fmt.Sprintf("%d:%+0.2f", a, v))
	}

	maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))
	var pressedButtons []string
	for b := ebiten.GamepadButton(id); b < maxButton; b++ {
		if ebiten.IsGamepadButtonPressed(id, b) {
			pressedButtons = append(pressedButtons, strconv.Itoa(int(b)))
		}
	}

	str += fmt.Sprintf("Gamepad (ID: %d, SDL ID: %s)%s:\n", id, ebiten.GamepadSDLID(id), standard)
	str += fmt.Sprintf("  Name:    %s\n", ebiten.GamepadName(id))
	str += fmt.Sprintf("  Axes:    %s\n", strings.Join(axes, ","))
	str += fmt.Sprintf("  Buttons: %s\n", strings.Join(pressedButtons, ","))

	str += "\n"
	if ebiten.IsStandardGamepadLayoutAvailable(id) {
		m := `       [FBL ]                    [FBR ]
       [FTL ]                    [FTR ]

       [LT  ]       [CC  ]       [RT  ]
    [LL  ][LR  ] [CL  ][CR  ] [RL  ][RR  ]
       [LB  ]                    [RB  ]
             [LS  ]       [RS  ]
`

		for b, str := range standardButtonToString {
			placeholder := "[" + str + strings.Repeat(" ", 4-len(str)) + "]"
			v := ebiten.StandardGamepadButtonValue(id, b)
			switch {
			case !ebiten.IsStandardGamepadButtonAvailable(id, b):
				m = strings.Replace(m, placeholder, "  --  ", 1)
			case ebiten.IsStandardGamepadButtonPressed(id, b):
				m = strings.Replace(m, placeholder, fmt.Sprintf("[%0.2f]", v), 1)
			default:
				m = strings.Replace(m, placeholder, fmt.Sprintf(" %0.2f ", v), 1)
			}
		}

		// TODO: Use ebiten.IsStandardGamepadAxisAvailable
		m += fmt.Sprintf("    Left Stick:  X: %+0.2f, Y: %+0.2f\n    Right Stick: X: %+0.2f, Y: %+0.2f\n--",
			ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal),
			ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical),
			ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickHorizontal),
			ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical))
		str += m
	}
	return str
}
