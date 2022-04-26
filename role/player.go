package role

import (
	"embed"
	"game/mapCreator/mapManage"
	"game/status"
	"game/tools"
	"image"

	"github.com/fzipp/texturepacker"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	OFFSETX int = -30
	OFFSETY int = -50
)

var (
	plist_sheet   *texturepacker.SpriteSheet
	plist_sheet_2 *texturepacker.SpriteSheet
	plist_png     *image.Paletted
	plist_png_2   *image.Paletted
	opS           *ebiten.DrawImageOptions
	op            *ebiten.DrawImageOptions
	newPath       []uint8
	turnLoop      uint8 = 0
)

type Player struct {
	X            float64                //玩家世界坐标X
	Y            float64                //玩家世界坐标Y
	State        uint8                  //玩家状态
	Direction    uint8                  //玩家当前方向
	OldDirection uint8                  //玩家旧的方向
	MouseX       int                    //鼠标X坐标
	MouseY       int                    //鼠标Y坐标
	SkillName    string                 //技能名称
	image        *embed.FS              //静态资源获取
	map_c        mapManage.MapInterface //地图
	status       *status.StatusManage   //状态
	hp           float64                //血
	mp           float64                //蓝
}

//创建玩家
func NewPlayer(x, y float64, state, dir uint8, mx, my int, images *embed.FS, m mapManage.MapInterface, s *status.StatusManage) *Player {
	opS = &ebiten.DrawImageOptions{}
	op = &ebiten.DrawImageOptions{}
	play := &Player{
		X:            x, //地图坐标X
		Y:            y, //地图坐标Y
		State:        state,
		Direction:    dir,
		OldDirection: dir,
		MouseX:       mx,
		MouseY:       my,
		SkillName:    "", //技能名字
		image:        images,
		map_c:        m,
		status:       s,
		hp:           100,
		mp:           100,
	}
	return play
}

//Load Images
func (p *Player) LoadImages() {

	//加载玩家素材第一部分
	plist, _ := p.image.ReadFile("resource/man/warrior/ba2.png")
	plist_json, _ := p.image.ReadFile("resource/man/warrior/ba2.json")
	plist_sheet, plist_png = tools.GetImageFromPlistPaletted(plist, plist_json)
	//加载玩家素材第二部分
	plist, _ = p.image.ReadFile("resource/man/warrior/ba2_act.png")
	plist_json, _ = p.image.ReadFile("resource/man/warrior/ba2_act.json")
	plist_sheet_2, plist_png_2 = tools.GetImageFromPlistPaletted(plist, plist_json)
	p.SetPlayerState(0, 0)
}

//设置玩家状态
func (p *Player) SetPlayerState(s, d uint8) {
	p.State = s
	if p.status.Flg {
		p.Direction = d
	}
}

//更新玩家旧的方向
func (p *Player) UpdateOldPlayerDir(d uint8) {
	p.OldDirection = d
}

//获取图片
func (p *Player) GetAnimator(flg, name string, block uint8) (*ebiten.Image, int, int) {
	if flg == "man" {
		//判断加载素材的第几部分
		if block == 1 {
			return ebiten.NewImageFromImage(plist_png.SubImage(plist_sheet.Sprites[name].Frame)), plist_sheet.Sprites[name].SpriteSourceSize.Min.X, plist_sheet.Sprites[name].SpriteSourceSize.Min.Y
		} else {
			return ebiten.NewImageFromImage(plist_png_2.SubImage(plist_sheet_2.Sprites[name].Frame)), plist_sheet_2.Sprites[name].SpriteSourceSize.Min.X, plist_sheet_2.Sprites[name].SpriteSourceSize.Min.Y
		}
	} else {
		return nil, 0, 0
	}
}

//暗黑破坏神 16方位 移动 鼠标控制
func (p *Player) GetMouseController(dir uint8) {
	if p.status.Flg {
		speed := 0.0
		//判断是否走路
		if p.status.IsWalk && (p.Direction != dir || p.State != tools.Walk) {
			speed = tools.SPEED
			p.SetPlayerState(tools.Walk, dir)
		}
		if !p.status.IsWalk && (p.Direction != dir || p.State != tools.RUN) {
			speed = tools.SPEED_RUN
			p.SetPlayerState(tools.RUN, dir)
		}
		//移动判断
		moveX, moveY := tools.CalculateSpeed(dir, speed)
		if p.CanWalk(moveX, moveY, dir) {
			p.status.MoveOffsetX += -moveX
			p.status.MoveOffsetY += -moveY
			p.Y += moveY
			p.X += moveX
		}
		p.status.Flg = false
	}
}

//判断是否可以行走
func (p *Player) CanWalk(xS, yS float64, dir uint8) bool {
	block1 := p.map_c.GetBlock1Aera()
	x, y := tools.GetFloorPositionAt(p.X+xS-110, p.Y+yS+70)
	if x >= p.status.ReadMapSizeWidth || y >= p.status.ReadMapSizeHeight || x < 0 || y < 0 {
		p.SetPlayerState(tools.IDLE, dir)
		return false
	}
	if block1[y][x].Img == nil || p.map_c.GetBlock1AeraUpdate(x, y) {
		return true
	} else {
		p.SetPlayerState(tools.IDLE, dir)
		return false
	}
}

//玩家移动
func (p *Player) PlayerMove(mouseX int, dir *uint8) {
	//鼠标人物移动控制
	if !p.status.OpenBag || p.status.OpenBag && mouseX <= tools.LAYOUTX/2 {
		//判断人物方位
		if p.OldDirection != p.Direction && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			if !p.status.CalculateEnd {
				newPath = tools.CalculateDirPath(p.OldDirection, p.Direction)
				p.status.CalculateEnd = true
			}
			if len(newPath) >= 3 {
				if turnLoop >= uint8(len(newPath)) {
					turnLoop = uint8(len(newPath) - 1)
					*dir = newPath[turnLoop]
					p.UpdateOldPlayerDir(p.Direction)
				} else {
					*dir = newPath[turnLoop]
				}
				turnLoop++
				p.SetPlayerState(tools.IDLE, *dir)
			} else {
				//直接切换方向
				p.status.CalculateEnd = false
				turnLoop = 0
				p.UpdateOldPlayerDir(p.Direction)
				p.GetMouseController(p.Direction)
			}
		} else {
			p.status.CalculateEnd = false
			turnLoop = 0
			p.GetMouseController(*dir)
		}
	}
}
