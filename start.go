// Copyright (C) 2020  Raziman

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/ztrue/tracerr"

	"github.com/issadarkthing/gomu/anko"
	"github.com/issadarkthing/gomu/hook"
	"github.com/issadarkthing/gomu/player"
)

// Panel is used to keep track of childrens in slices
type Panel interface {
	HasFocus() bool
	SetBorderColor(color tcell.Color) *tview.Box
	SetTitleColor(color tcell.Color) *tview.Box
	SetTitle(s string) *tview.Box
	GetTitle() string
	help() []string
}

// Args is the args for gomu executable
type Args struct {
	config  *string
	empty   *bool
	music   *string
	version *bool
}

func getArgs() Args {
	cfd, err := os.UserConfigDir()
	if err != nil {
		logError(tracerr.Wrap(err))
	}
	configPath := filepath.Join(cfd, "gomu", "config")
	configFlag := flag.String("config", configPath, "Specify config file")
	emptyFlag := flag.Bool("empty", false, "Open gomu with empty queue. Does not override previous queue")
	home, err := os.UserHomeDir()
	if err != nil {
		logError(tracerr.Wrap(err))
	}
	musicPath := filepath.Join(home, "Music")
	musicFlag := flag.String("music", musicPath, "Specify music directory")
	versionFlag := flag.Bool("version", false, "Print gomu version")
	flag.Parse()
	return Args{
		config:  configFlag,
		empty:   emptyFlag,
		music:   musicFlag,
		version: versionFlag,
	}
}

// built-in functions
func defineBuiltins() {
	gomu.anko.DefineGlobal("debug_popup", debugPopup)
	gomu.anko.DefineGlobal("info_popup", infoPopup)
	gomu.anko.DefineGlobal("input_popup", inputPopup)
	gomu.anko.DefineGlobal("show_popup", defaultTimedPopup)
	gomu.anko.DefineGlobal("search_popup", searchPopup)
	gomu.anko.DefineGlobal("shell", shell)
}

func defineInternals() {
	playlist, _ := gomu.anko.NewModule("Playlist")
	playlist.Define("get_focused", gomu.playlist.getCurrentFile)
	playlist.Define("focus", func(filepath string) {

		root := gomu.playlist.GetRoot()
		root.Walk(func(node, _ *tview.TreeNode) bool {

			if node.GetReference().(*player.AudioFile).Path() == filepath {
				gomu.playlist.setHighlight(node)
				return false
			}

			return true
		})
	})

	queue, _ := gomu.anko.NewModule("Queue")
	queue.Define("get_focused", func() *player.AudioFile {
		index := gomu.queue.GetCurrentItem()
		if index < 0 || index > len(gomu.queue.items)-1 {
			return nil
		}
		item := gomu.queue.items[index]
		return item
	})

	player, _ := gomu.anko.NewModule("Player")
	player.Define("current_audio", gomu.player.GetCurrentSong)
}

func setupHooks(hook *hook.EventHook, anko *anko.Anko) {

	events := []string{
		"enter",
		"new_song",
		"skip",
		"play",
		"pause",
		"exit",
	}

	for _, event := range events {
		name := event
		hook.AddHook(name, func() {
			src := fmt.Sprintf(`Event.run_hooks("%s")`, name)
			_, err := anko.Execute(src)
			if err != nil {
				err = tracerr.Errorf("error execute hook: %w", err)
				logError(err)
			}
		})
	}
}

// loadModules executes helper modules and default config that should only be
// executed once
func loadModules(env *anko.Anko) error {

	const listModule = `
module List {

	func collect(l, f) {
		result = []
		for x in l {
			result += f(x)
		}
		return result
	}

	func filter(l, f) {
		result = []
		for x in l {
			if f(x) {
				result += x
			}
		}
		return result
	}

	func reduce(l, f, acc) {
		for x in l {
			acc = f(acc, x)
		}
		return acc
	}
}
`
	const eventModule = `
module Event {
	events = {}

	func add_hook(name, f) {
		hooks = events[name]

		if hooks == nil {
			events[name] = [f]
			return
		}

		hooks += f
		events[name] = hooks
	}

	func run_hooks(name) {
		hooks = events[name]

		if hooks == nil {
			return
		}

		for hook in hooks {
			hook()
		}
	}
}
`

	const keybindModule = `
module Keybinds {
	global = {}
	playlist = {}
	queue = {}

	func def_g(kb, f) {
		global[kb] = f
	}

	func def_p(kb, f) {
		playlist[kb] = f
	}

	func def_q(kb, f) {
		queue[kb] = f
	}
}
`
	_, err := env.Execute(eventModule + listModule + keybindModule)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

type Config struct {
	General struct {
		confirm_bulk_add   bool     `yaml:"confirm_bulk_add"`
		confirm_on_exit    bool     `yaml:"confirm_on_exit"`
		queue_loop         bool     `yaml:"queue_loop"`
		load_prev_queue    bool     `yaml:"load_prev_queue"`
		popup_timeout      int16    `yaml:"popup_timeout"`
		sort_by_mtime      bool     `yaml:"sort_by_mtime"`
		music_dirs         []string `yaml:",flow"`
		history_path       string   `yaml:"history_path"`
		use_emoji          bool     `yaml:"use_emoji"`
		volume             int16    `yaml:"volume"`
		invidious_instance string   `yaml:"invidious_instance"`
		lang_lyrics        []string `yaml:"lang_lyrics":,flow`
		rename_bytag       bool     `yaml:"rename_bytag"`
	} `yaml:"general"`
	Emoji struct {
		playlist string `yaml:"playlist"`
		file     string `yaml:"file"`
		loop     string `yaml:"loop"`
		shuffle  string `yaml:"shuffle"`
		noloop   string `yaml:"noloop"`
	} `yaml:"emoji"`
	// TODO: use CSS
	Color struct {
		accent     string `yaml:"accent"`
		background string `yaml:"background"`
		foreground string `yaml:"foreground"`
		popup      string `yaml:"popup"`

		playlist struct {
			directory string `yaml:"directory"`
			highlight string `yaml:"highlight"`
		} `yaml:"playlist"`

		queue_highlight string `yaml:"queue_highlight"`

		now_playing struct {
			artist string `yaml:"artist"`
			title  string `yaml:"title"`
			album  string `yaml:"album"`
		} `yaml:"now_playing"`

		subtitle string `yaml:"subtitle"`
	} `yaml:"color"`
}

// executes user config with default config is executed first in order to apply
// default values
func execConfig(config string) error {

	const defaultConfig = `
	general:
		# Confirmation popup for adding the whole playlist to the queue
		confirm_bulk_add: true
		confirm_on_exit: true
		queue_loop: false
		load_prev_queue: true
		popup_timeout: 5
		sort_by_mtime: false
		music_dirs: ["$HOME/Music"]
		# url history of downloaded audio will be saved here
		history_path: $HOME/.local/share/gomu/urls
		# Some terminals support unicode characters, set this to false if you want to use ascii characters instead
		use_emoji: true
		volume: 80	
		# See https://github.com/iv-org/documentation/blob/master/Invidious-Instances.md
		invidious_instance: "https://vid.puffyan.us"
		lang_lyrics: ["en"]
		# When saving tags, rename the the file as per the tags (artist-songname-album)
		rename_bytag: false
	emoji:
		playlist: "📁"
		file: "🎵"
		loop: "🔁"
		noloop: ""
		shuffle: "🔀"
	color:
		#you may change colors by pressing 'c'
		accent: "darkcyan"
		background: "none"
		foreground: "white"
		popup: "black"
		playlist:
			directory: "darkcyan"
			highlight: "darkcyan"
		queue_highlight: "darkcyan"
		now_playing:
			artist: "lightgreen"
			title: "green"
			album: "darkgreen"
		subtitle: "darkgoldenrod"
	`
	// if the function is not called by defineCommands, then the config is Args.config
	args := getArgs()
	if config == "" {
		config = *args.config
	}

	cfg := expandTilde(config)

	_, err := os.Stat(cfg)
	if os.IsNotExist(err) {
		err = appendFile(cfg, defaultConfig)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	content, err := ioutil.ReadFile(cfg)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute default config
	_, err = gomu.anko.Execute(defaultConfig)
	if err != nil {
		return tracerr.Wrap(err)
	}

	// execute user config
	_, err = gomu.anko.Execute(string(content))
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

// Sets the layout of the application
func layout(gomu *Gomu) *tview.Flex {
	flex := tview.NewFlex().
		AddItem(gomu.playlist, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(gomu.queue, 0, 5, false).
			AddItem(gomu.playingBar, 9, 0, false), 0, 2, false)

	return flex
}

// Initialize
func start(application *tview.Application, args Args) {

	// Print version and exit
	if *args.version {
		fmt.Printf("Gomu %s\n", VERSION)
		return
	}

	// Assigning to global variable gomu
	gomu = newGomu()
	gomu.command.defineCommands()
	defineBuiltins()

	err := loadModules(gomu.anko)
	if err != nil {
		die(err)
	}

	err = execConfig(expandFilePath(*args.config))
	if err != nil {
		die(err)
	}

	setupHooks(gomu.hook, gomu.anko)

	gomu.hook.RunHooks("enter")
	gomu.args = args
	gomu.colors = newColor()

	// override default border
	// change double line border to one line border when focused
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.PrimitiveBackgroundColor = gomu.colors.popup

	gomu.initPanels(application, args)
	defineInternals()

	gomu.player.SetSongStart(func(audio player.Audio) {

		duration, err := getTagLength(audio.Path())
		if err != nil || duration == 0 {
			duration, err = player.GetLength(audio.Path())
			if err != nil {
				logError(err)
				return
			}
		}

		audioFile := audio.(*player.AudioFile)

		gomu.playingBar.newProgress(audioFile, int(duration.Seconds()))

		name := audio.Name()
		var description string

		if len(gomu.playingBar.subtitles) == 0 {
			description = name
		} else {
			lang := gomu.playingBar.subtitle.LangExt

			description = fmt.Sprintf("%s \n\n %s lyric loaded", name, lang)
		}

		defaultTimedPopup(" Now Playing ", description)

		go func() {
			err := gomu.playingBar.run()
			if err != nil {
				logError(err)
			}
		}()

	})

	gomu.player.SetSongFinish(func(currAudio player.Audio) {

		gomu.playingBar.subtitles = nil
		var mu sync.Mutex
		mu.Lock()
		gomu.playingBar.subtitle = nil
		mu.Unlock()
		if gomu.queue.isLoop {
			_, err = gomu.queue.enqueue(currAudio.(*player.AudioFile))
			if err != nil {
				logError(err)
			}
		}

		if len(gomu.queue.items) > 0 {
			err := gomu.queue.playQueue()
			if err != nil {
				logError(err)
			}
		} else {
			gomu.playingBar.setDefault()
		}
	})

	flex := layout(gomu)
	gomu.pages.AddPage("main", flex, true, true)

	// sets the first focused panel
	gomu.setFocusPanel(gomu.playlist)
	gomu.prevPanel = gomu.playlist

	gomu.playingBar.setDefault()

	gomu.queue.isLoop = gomu.anko.GetBool("General.queue_loop")

	loadQueue := gomu.anko.GetBool("General.load_prev_queue")

	if !*args.empty && loadQueue {
		// load saved queue from previous session
		if err := gomu.queue.loadQueue(); err != nil {
			logError(err)
		}
	}

	if len(gomu.queue.items) > 0 {
		if err := gomu.queue.playQueue(); err != nil {
			logError(err)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		errMsg := fmt.Sprintf("Received %s. Exiting program", sig.String())
		logError(errors.New(errMsg))
		err := gomu.quit(args)
		if err != nil {
			logError(errors.New("unable to quit program"))
		}
	}()

	cmds := map[rune]string{
		'q': "quit",
		' ': "toggle_pause",
		'+': "volume_up",
		'=': "volume_up",
		'-': "volume_down",
		'_': "volume_down",
		'n': "skip",
		':': "command_search",
		'?': "toggle_help",
		'f': "forward",
		'F': "forward_fast",
		'b': "rewind",
		'B': "rewind_fast",
		'm': "repl",
		'T': "switch_lyric",
		'c': "show_colors",
	}

	for key, cmdName := range cmds {
		src := fmt.Sprintf(`Keybinds.def_g("%c", %s)`, key, cmdName)
		gomu.anko.Execute(src)
	}

	// global keybindings are handled here
	application.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {

		if gomu.pages.HasPage("repl-input-popup") {
			return e
		}

		if gomu.pages.HasPage("tag-editor-input-popup") {
			return e
		}

		popupName, _ := gomu.pages.GetFrontPage()

		// disables keybindings when writing in input fields
		if strings.Contains(popupName, "-input-") {
			return e
		}

		switch e.Key() {
		// cycle through each section
		case tcell.KeyTAB:
			if strings.Contains(popupName, "confirmation-") {
				return e
			}
			gomu.cyclePanels2()
		}

		if gomu.anko.KeybindExists("global", e) {

			err := gomu.anko.ExecKeybind("global", e)
			if err != nil {
				errorPopup(err)
			}

			return nil
		}

		return e
	})

	// fix transparent background issue
	gomu.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	init := false
	gomu.app.SetAfterDrawFunc(func(_ tcell.Screen) {
		if !init && len(gomu.queue.items) == 0 {
			gomu.playingBar.setDefault()
			init = true
		}
	})

	gomu.app.SetRoot(gomu.pages, true).SetFocus(gomu.playlist)

	// main loop
	if err := gomu.app.Run(); err != nil {
		die(err)
	}

	gomu.hook.RunHooks("exit")
}
