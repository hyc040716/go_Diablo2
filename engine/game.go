package engine

import (
	"embed"
	"fmt"
	"game/layout"
	"game/mapCreator/anm"
	"game/maps"
	"game/music"
	"game/role"
	"game/status"
	"game/tools"
	"math"
	"runtime"
	"strconv"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//config
const (
	SCREENWIDTH  int = 490
	SCREENHEIGHT int = 300

	WEOFFSETX           int = 127
	WEOFFSETY           int = 14
	GAMESCENELOGIN      int = 1
	GAMESCENESELECTROLE int = 2
	GAMESCENEOPENDOOR   int = 3
	GAMESCENESTART      int = 4
)

type Game struct {
	count             int
	countForMap       int
	player            *role.Player  //玩家
	maps              *maps.MapBase //地图
	objectA           *anm.Anm      //object 动画
	ui                *layout.UI    //UI
	currentGameScence int
	music             music.MusicInterface //音乐
	status            *status.StatusManage //状态管理器
}

var (
	counts       int = 0
	countsForMap int = 0
	frameNums    int = 4
	frameSpeed   int = 5
	images       *embed.FS

	mouseX   int
	mouseY   int
	newPath  []uint8
	turnLoop uint8 = 0
)

//GameEngine
func NewGame(asset *embed.FS) *Game {
	images = asset
	//statueManage
	sta := status.NewStatusManage()
	//Map
	m := maps.NewMap(asset, sta)
	//Player
	r := role.NewPlayer(0, 0, tools.IDLE, 0, 0, 0, asset, m, sta)
	//UI
	u := layout.NewUI(asset, sta, m)

	//BGM
	bgm := music.NewMusicBGM(images)

	//object Anm
	object := anm.NewAnm(asset, sta)

	gameEngine := &Game{
		count:             0,
		countForMap:       0,
		player:            r,
		maps:              m,
		ui:                u,
		currentGameScence: GAMESCENELOGIN,
		music:             bgm,
		status:            sta,
		objectA:           object,
	}
	return gameEngine
}

func (g *Game) StartEngine() {
	//隐藏鼠标系统的ICON
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	//
	w := sync.WaitGroup{}
	w.Add(1)
	//UI Init
	go func() {
		g.ui.LoadGameLoginImages()
		runtime.GC()
		w.Done()
	}()
	w.Wait()
	go func() {
		runtime.GC()
	}()
}

//Change Load
func (g *Game) ChangeScene(name string) {
	if name == "select" {
		g.currentGameScence = GAMESCENESELECTROLE
		g.ui.LoadGameCharaSelectImages()
		runtime.GC()

	} else if name == "loading" {
		g.currentGameScence = GAMESCENEOPENDOOR
		//GC
		g.ui.ClearSlice(0)
		g.ui.ClearGlobalVariable()

	} else if name == "game" {
		g.currentGameScence = GAMESCENESTART
		w := sync.WaitGroup{}
		w.Add(3)
		//Palyer Init
		go func() {
			g.player.LoadImages()
			runtime.GC()
			w.Done()
		}()
		//Ui
		go func() {
			g.ui.LoadGameImages()
			runtime.GC()
			w.Done()
		}()
		//Map Init
		go func() {
			g.maps.LoadMap()
			//加载动画
			g.objectA.LoadAnm()
			runtime.GC()
			w.Done()
		}()
		w.Wait()
		go func() {
			runtime.GC()
		}()
	}

}
func (g *Game) Update() error {
	mouseX, mouseY = ebiten.CursorPosition()
	//切换场景判定
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !g.status.ChangeScenceFlg {
		if g.currentGameScence == GAMESCENELOGIN {
			if mouseX > 286 && mouseX < 503 && mouseY > 150 && mouseY < 218 {
				g.currentGameScence = GAMESCENESELECTROLE
				g.status.ChangeScenceFlg = true
				w := sync.WaitGroup{}
				w.Add(1)
				go func() {
					g.ChangeScene("select")
					w.Done()
				}()
				w.Wait()
				g.status.ChangeScenceFlg = false
			}

		} else if g.currentGameScence == GAMESCENESELECTROLE {
			if mouseX > 684 && mouseX < 759 && mouseY > 390 && mouseY < 470 {
				g.status.ChangeScenceFlg = true
				g.currentGameScence = GAMESCENEOPENDOOR
				g.status.ChangeScenceFlg = false
			}
		}
	}

	//切换场景逻辑
	if !g.status.ChangeScenceFlg {
		if g.currentGameScence == GAMESCENESTART {
			//进入游戏场景逻辑
			g.changeScenceGameUpdate()
		} else if g.currentGameScence == GAMESCENEOPENDOOR {
			//游戏加载逻辑
			g.ChangeScenceOpenDoorUpdate()
		} else {
			//进入游戏登录界面逻辑
			g.ChangeScenceLoginUpdate()
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	//判断是否切换场景
	if !g.status.ChangeScenceFlg {
		if g.currentGameScence == GAMESCENESTART {
			g.ChangeScenceGameDraw(screen)
		} else if g.currentGameScence == GAMESCENESELECTROLE {
			g.ChangeScenceSelectDraw(screen)
		} else if g.currentGameScence == GAMESCENEOPENDOOR {
			g.ChangeScenceOpenDoorDraw(screen)
		} else {
			g.ChangeScenceLoginDraw(screen)
		}
	}
	//绘制鼠标ICON
	g.ui.DrawMouseIcon(screen, mouseX, mouseY)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return tools.LAYOUTX, tools.LAYOUTY
}

//Draw Game Update
func (g *Game) changeScenceGameUpdate() {
	g.count++
	g.countForMap++
	if !g.status.MusicIsPlay {
		//音乐
		g.status.MusicIsPlay = true
		g.music.PlayMusic("Bar_act2_complete_tombs.wav", tools.MUSICWAV)
	}
	if g.player.State != tools.ATTACK {
		g.player.State = tools.IDLE
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.player.MouseX = mouseX
		g.player.MouseY = mouseY
		//防止点击UI界面也移动
		if mouseY < 436 {
			g.status.Flg = true
		}
		//如果打开包裹，包裹已右位置不能点击移动
		if g.status.OpenBag && mouseX >= tools.LAYOUTX/2 {
			g.status.Flg = false
		}
		//如果打开MINi板子，并且没有打开包裹 以下坐标不可以点击移动
		if g.status.OpenMiniPanel && !g.status.OpenBag && mouseX >= 305 && mouseX <= 475 && mouseY > 407 {
			g.status.Flg = false
		}
		//如果打开MINi板子，并且打开包裹 以下坐标不可以点击移动
		if g.status.OpenMiniPanel && g.status.OpenBag && mouseX >= 205 && mouseX <= 377 && mouseY > 407 {
			g.status.Flg = false
		}
		//如果拿起物品也不可以移动
		if g.status.IsTakeItem {
			g.status.Flg = false
		}
	}

	//计算鼠标位置
	dir := tools.CaluteDir(g.status.PLAYERCENTERX, g.status.PLAYERCENTERY, int64(g.player.MouseX), int64(g.player.MouseY))

	//TODO 技能攻击
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && !g.status.IsTakeItem {
		g.player.SkillName = "liehuo"
		if g.player.State != tools.ATTACK {
			counts = 0
		}
		g.status.Flg = false
		if g.player.Direction != dir || g.player.State != tools.ATTACK {
			g.player.SetPlayerState(tools.ATTACK, dir)

		}
	}
	//事件循环监听 是否有按钮点击事件
	g.ui.EventLoop(mouseX, mouseY)
	//鼠标移动控制
	if !g.status.OpenBag || g.status.OpenBag && mouseX <= tools.LAYOUTX/2 {
		//
		if g.player.OldDirection != g.player.Direction && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			if !g.status.CalculateEnd {
				newPath = tools.CalculateDirPath(g.player.OldDirection, g.player.Direction)
				g.status.CalculateEnd = true
			}
			if len(newPath) >= 3 {
				if turnLoop >= uint8(len(newPath)) {
					turnLoop = uint8(len(newPath) - 1)
					dir = newPath[turnLoop]
					g.player.UpdateOldPlayerDir(g.player.Direction)
				} else {
					dir = newPath[turnLoop]
				}
				turnLoop++
				g.player.SetPlayerState(tools.IDLE, dir)
			} else {
				g.status.CalculateEnd = false
				turnLoop = 0
				g.player.UpdateOldPlayerDir(g.player.Direction)
				g.player.GetMouseController(dir)
			}

		} else {
			g.status.CalculateEnd = false
			turnLoop = 0
			g.player.UpdateOldPlayerDir(g.player.Direction)
			g.player.GetMouseController(dir)
		}

	}
	//根据状态改变帧数
	if g.player.State == tools.IDLE {
		frameNums = 16
		frameSpeed = 5
		g.player.SetPlayerState(tools.IDLE, dir)

	} else if g.player.State == tools.ATTACK {
		frameNums = 16
		frameSpeed = 1
	} else {
		frameNums = 8
		frameSpeed = 5
	}

}

//Draw Game Scence  渲染游戏画面
func (g *Game) ChangeScenceGameDraw(screen *ebiten.Image) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("has error is :", r)
		}
	}()
	//Draw floor
	g.maps.RenderFloor(screen, g.status.MoveOffsetX, g.status.MoveOffsetY)
	//
	g.player.Render(screen, counts)
	//Draw Wall
	g.maps.RenderWall(screen, g.status.MoveOffsetX, g.status.MoveOffsetY)
	//Draw object Anmi
	g.objectA.Render(screen, countsForMap, g.status.MoveOffsetX, g.status.MoveOffsetY)
	//Draw UI
	g.ui.DrawUI(screen)

	//Draw Debug
	len := tools.Distance(g.status.PLAYERCENTERX, g.status.PLAYERCENTERY, int64(mouseX), int64(mouseY))
	re := tools.Angle(math.Abs(float64(int64(mouseY)-g.status.PLAYERCENTERY)), len)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS %d\nplayer position %d,%d\nmouse position %d,%d\ndir %d\nAngle %f",
		int64(ebiten.CurrentFPS()), int64(g.player.X), int64(g.player.Y), g.player.MouseX, g.player.MouseY, tools.CaluteDir(g.status.PLAYERCENTERX, g.status.PLAYERCENTERY, int64(g.player.MouseX), int64(g.player.MouseY)), re))

	//Change player Frame
	if g.count > frameSpeed {
		counts++
		g.count = 0
		if counts >= frameNums {
			counts = 0
		}
	}

	//Change map Frame
	if g.countForMap > 5 {
		countsForMap++
		g.countForMap = 0
		if countsForMap >= 20 {
			countsForMap = 0
		}
	}
}

//Draw Login Update
func (g *Game) ChangeScenceLoginUpdate() {
	if !g.status.MusicIsPlay {
		g.status.MusicIsPlay = true
		g.music.PlayMusic("Act0-Intro.mp3", tools.MUSICMP3)
	}
	frameSpeed_clone := 0
	if g.currentGameScence == GAMESCENELOGIN {
		frameSpeed_clone = 2
	} else {
		frameSpeed_clone = 5
	}
	g.count++
	//Change Frame
	if g.count > frameSpeed_clone {
		counts++
		g.count = 0
		if counts >= 30 {
			counts = 0
		}
	}
}

//Draw Login Scence 渲染游戏登录画面
func (g *Game) ChangeScenceLoginDraw(screen *ebiten.Image) {

	//Draw UI
	g.ui.DrawUI(screen)

	//Draw Logo Left
	name := "logoFireLeft_" + strconv.Itoa(counts) + ".png"
	left, _, _ := g.ui.GetAnimator("logo", name)
	opLo := &ebiten.DrawImageOptions{}
	opLo.Filter = ebiten.FilterLinear
	opLo.GeoM.Translate(220, 0)
	opLo.CompositeMode = ebiten.CompositeModeLighter
	opLo.GeoM.Scale(1, 0.7)
	screen.DrawImage(left, opLo)
	//Draw Logo Right
	name = "logoFireRight_" + strconv.Itoa(counts) + ".png"
	right, _, _ := g.ui.GetAnimator("logo", name)
	opRo := &ebiten.DrawImageOptions{}
	opRo.Filter = ebiten.FilterLinear
	opRo.GeoM.Translate(float64(220+right.Bounds().Max.X), 0)
	opRo.CompositeMode = ebiten.CompositeModeLighter
	opRo.GeoM.Scale(1, 0.7)
	screen.DrawImage(right, opRo)
	//Draw Debug
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS %d\nmouse position %d,%d",
		int64(ebiten.CurrentFPS()), mouseX, mouseY))

	//Change Frame
	if g.count > frameSpeed {
		counts++
		g.count = 0
		if counts >= 30 {
			counts = 0
		}
	}
}

//Draw Select Scence  渲染游戏角色选择画面
func (g *Game) ChangeScenceSelectDraw(screen *ebiten.Image) {
	//Draw UI
	g.ui.DrawUI(screen)

	//Draw Fire
	name := "fire_" + strconv.Itoa(counts) + ".png"
	fire, _, _ := g.ui.GetAnimator("logo", name)
	opf := &ebiten.DrawImageOptions{}
	opf.Filter = ebiten.FilterLinear
	opf.GeoM.Translate(350, 375)
	opf.CompositeMode = ebiten.CompositeModeLighter
	opf.GeoM.Scale(1, 0.7)
	screen.DrawImage(fire, opf)

	//Draw Role Ba
	name = "ba_" + strconv.Itoa(counts) + ".png"
	ba, _, _ := g.ui.GetAnimator("role", name)
	opBa := &ebiten.DrawImageOptions{}
	opBa.Filter = ebiten.FilterLinear
	opBa.GeoM.Translate(356, 120)
	screen.DrawImage(ba, opBa)

	//Draw Debug
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS %d\nmouse position %d,%d", int64(ebiten.CurrentFPS()), mouseX, mouseY))
}

//Draw OpenDoor Scence 渲染游戏加载开门画面
func (g *Game) ChangeScenceOpenDoorDraw(screen *ebiten.Image) {
	//Draw Open Door
	name := "loading_" + strconv.Itoa(counts) + ".png"
	door, _, _ := g.ui.GetAnimator("logo", name)
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(268, 120)
	screen.DrawImage(door, op)

	//Draw Debug
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS %d\nmouse position %d,%d", int64(ebiten.CurrentFPS()), mouseX, mouseY))
}

//Draw OpenDoor Update
func (g *Game) ChangeScenceOpenDoorUpdate() {
	if !g.status.DoorCountFlg {
		counts = 0
		g.status.DoorCountFlg = true
	}
	g.count++
	//Change Frame
	if g.count > 10 && counts != 9 {
		counts++
		g.count = 0
	}

	// Change Scence
	if counts == 9 && !g.status.LoadingFlg {
		g.status.LoadingFlg = true
		g.currentGameScence = GAMESCENESTART
		w := sync.WaitGroup{}
		w.Add(1)
		go func() {
			//close music
			g.status.MusicIsPlay = false
			g.music.CloseMusic()
			g.ChangeScene("game")
			w.Done()
		}()
		w.Wait()
		g.status.ChangeScenceFlg = false
	}
}
