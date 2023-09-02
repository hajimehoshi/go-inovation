//go:build !js && steam

package lang

import (
	"os"

	"github.com/hajimehoshi/go-steamworks"
	"golang.org/x/text/language"
)

const appID = 1710390

func init() {
	if steamworks.RestartAppIfNecessary(appID) {
		os.Exit(1)
	}
	if !steamworks.Init() {
		panic("steamworks.Init failed")
	}
}

func SystemLang() language.Tag {
	switch steamworks.SteamApps().GetCurrentGameLanguage() {
	case "english":
		return language.English
	case "japanese":
		return language.Japanese
	}
	return language.English
}
