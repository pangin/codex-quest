package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Type alias for cleaner pattern definitions.
type C = color.RGBA

const (
	frameWidth  = 32
	frameHeight = 32
	numAnims    = 10
	maxFrames   = 24
)

// Animation frame counts (must match animations.go).
// Idle, Enter, Casting, Attack, Writing, Victory, Hurt, Thinking, Walk, VictoryPose
var frameCounts = []int{16, 20, 16, 16, 16, 20, 16, 16, 16, 20}

// Codex Pet-inspired palette. The short names are shared by the accessory
// generators, so keep them stable even though the colors are now blue.
var (
	P = C{0x5D, 0x8D, 0xFF, 0xFF} // Primary Codex blue
	S = C{0x31, 0x4B, 0xC9, 0xFF} // Shadow blue
	H = C{0x9D, 0xBC, 0xFF, 0xFF} // Soft highlight blue
	O = C{0x0B, 0x16, 0x38, 0xFF} // Deep navy outline/panel
	M = C{0x15, 0x2A, 0x66, 0xFF} // Mid navy detail
	W = C{0xF3, 0xFA, 0xFF, 0xFF} // Cool white
	G = C{0x72, 0xF6, 0xFF, 0xFF} // Terminal cyan
	Y = C{0xFF, 0xEA, 0x82, 0xFF} // Spark yellow
	X = C{0x00, 0x00, 0x00, 0x00} // Transparent

	panelFill = C{0x10, 0x1E, 0x4A, 0xFF}
	panelHi   = C{0x25, 0x3B, 0x7E, 0xFF}
	glowBlue  = C{0xB8, 0xF8, 0xFF, 0xFF}
)

type petFace int

const (
	facePrompt petFace = iota
	faceBlink
	faceHappy
	faceBars
	faceX
	faceWorry
	faceSquint
)

type armPose int

const (
	armsIdle armPose = iota
	armsUp
	armsCast
	armsPunch
	armsTyping
	armsHurt
	armsProud
)

type petPose struct {
	X, Y    int
	SquashX int
	SquashY int
	Face    petFace
	Arms    armPose
	Legs    int
	Laptop  bool
}

func createImage(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

func main() {
	width := frameWidth * maxFrames
	height := frameHeight * numAnims
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, X)
		}
	}

	for anim := 0; anim < numAnims; anim++ {
		for frame := 0; frame < frameCounts[anim]; frame++ {
			drawFrame(img, anim, frame)
		}
	}

	os.MkdirAll("assets/codex", 0755)
	f, _ := os.Create("assets/codex/spritesheet.png")
	defer f.Close()
	png.Encode(f, img)

	generateAccessoriesWithLegacyPalette()
	generateMiniCodex()
	generateEnemies()
	generateChest()
}

func generateAccessoriesWithLegacyPalette() {
	current := []C{P, S, H, O, M, W, G, Y}
	P = C{0xFF, 0x99, 0x33, 0xFF}
	S = C{0xCC, 0x66, 0x00, 0xFF}
	H = C{0xFF, 0xBB, 0x77, 0xFF}
	O = C{0x22, 0x22, 0x22, 0xFF}
	M = C{0x44, 0x22, 0x00, 0xFF}
	W = C{0xFF, 0xFF, 0xFF, 0xFF}
	G = C{0x00, 0xFF, 0x88, 0xFF}
	Y = C{0xFF, 0xF5, 0x96, 0xFF}

	generateAccessories()

	P, S, H, O, M, W, G, Y = current[0], current[1], current[2], current[3], current[4], current[5], current[6], current[7]
}

func drawFrame(img *image.RGBA, anim, frame int) {
	offsetX := frame * frameWidth
	offsetY := anim * frameHeight

	switch anim {
	case 0:
		drawCodexIdle(img, offsetX, offsetY, frame)
	case 1:
		drawCodexEnter(img, offsetX, offsetY, frame)
	case 2:
		drawCodexCasting(img, offsetX, offsetY, frame)
	case 3:
		drawCodexAttack(img, offsetX, offsetY, frame)
	case 4:
		drawCodexWriting(img, offsetX, offsetY, frame)
	case 5:
		drawCodexVictory(img, offsetX, offsetY, frame)
	case 6:
		drawCodexHurt(img, offsetX, offsetY, frame)
	case 7:
		drawCodexThinking(img, offsetX, offsetY, frame)
	case 8:
		drawCodexWalk(img, offsetX, offsetY, frame)
	case 9:
		drawCodexVictoryPose(img, offsetX, offsetY, frame)
	}
}

func defaultPose() petPose {
	return petPose{Face: facePrompt, Arms: armsIdle, Legs: 0}
}

func drawCodexIdle(img *image.RGBA, ox, oy, frame int) {
	bob := []int{0, 0, 0, -1, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, 0, 0}
	pose := defaultPose()
	pose.Y = bob[frame%len(bob)]
	if frame >= 12 && frame <= 14 {
		pose.Face = faceBlink
	}
	drawCodexPetBase(img, ox, oy, pose)
}

func drawCodexEnter(img *image.RGBA, ox, oy, frame int) {
	if frame < 5 {
		drawSparkle(img, ox+16, oy+17-frame, Y)
		drawSparkle(img, ox+10+frame, oy+22-frame/2, W)
		drawSparkle(img, ox+22-frame, oy+20-frame/2, G)
		return
	}
	if frame < 8 {
		y := oy + 22 - (frame-5)*2
		drawShadedRoundedRect(img, ox+13, y, 6, 4, P)
		drawSparkle(img, ox+8, oy+14, G)
		drawSparkle(img, ox+24, oy+12, Y)
		return
	}

	pose := defaultPose()
	if frame < 15 {
		settleY := []int{3, 2, 1, 0, -1, 0, 0}
		squashX := []int{2, 1, 1, 0, 0, 0, 0}
		squashY := []int{-1, 0, 0, 0, 1, 0, 0}
		idx := frame - 8
		pose.Y = settleY[idx]
		pose.SquashX = squashX[idx]
		pose.SquashY = squashY[idx]
	} else {
		bounce := []int{-2, -1, 0, 0, 0}
		pose.Y = bounce[frame-15]
	}
	drawCodexPetBase(img, ox, oy, pose)
	if frame < 12 {
		drawSparkle(img, ox+6, oy+9, Y)
		drawSparkle(img, ox+26, oy+12, G)
	}
}

func drawCodexCasting(img *image.RGBA, ox, oy, frame int) {
	pose := defaultPose()
	pose.Face = faceBars
	pose.Arms = armsCast

	if frame < 5 {
		pose.Y = []int{2, 2, 1, 0, -1}[frame]
		pose.SquashX = []int{1, 1, 0, 0, 0}[frame]
		pose.SquashY = []int{-1, -1, 0, 0, 1}[frame]
	} else if frame < 13 {
		pose.Y = []int{-2, -3, -3, -2, -2, -3, -2, -2}[frame-5]
	} else {
		pose.Y = []int{-1, 0, 0}[frame-13]
		pose.Arms = armsIdle
	}

	drawCodexPetBase(img, ox, oy, pose)
	if frame >= 5 && frame < 13 {
		phase := frame - 5
		for i := 0; i < 6; i++ {
			angle := (phase*30 + i*60) % 360
			x := ox + 16 + simpleCos(angle)*10/100
			y := oy + 8 + pose.Y + simpleSin(angle)*8/100
			if (i+frame)%2 == 0 {
				drawSparkle(img, x, y, G)
			} else {
				drawSparkle(img, x, y, Y)
			}
		}
	}
}

func drawCodexAttack(img *image.RGBA, ox, oy, frame int) {
	if frame == 7 {
		drawCodexPetSmear(img, ox, oy)
		drawImpactBurst(img, ox+30, oy+15)
		return
	}

	pose := defaultPose()
	pose.Face = faceSquint
	pose.Arms = armsPunch
	switch {
	case frame < 3:
		pose.Y = []int{1, 2, 2}[frame]
		pose.X = -frame
		pose.SquashX = 1
		pose.SquashY = -1
	case frame < 5:
		pose.Y = 3
		pose.X = -2
		pose.SquashX = 2
		pose.SquashY = -1
	case frame < 7:
		pose.Y = []int{2, 1}[frame-5]
	case frame < 10:
		pose.X = 2
		pose.Y = 1
	case frame < 14:
		pose.X = []int{1, 1, 0, 0}[frame-10]
		pose.Y = []int{-1, 0, 1, 0}[frame-10]
		pose.Arms = armsIdle
	default:
		pose.Y = []int{-1, 0}[frame-14]
		pose.Face = facePrompt
		pose.Arms = armsIdle
	}

	drawCodexPetBase(img, ox, oy, pose)
	if frame >= 8 && frame < 11 {
		drawImpactBurst(img, ox+30, oy+15)
	}
}

func drawCodexWriting(img *image.RGBA, ox, oy, frame int) {
	bob := []int{0, -1, -1, 0, 0, -1, -1, 0, 0, -1, -1, 0, 0, -1, 0, 0}
	pose := defaultPose()
	pose.Y = bob[frame%len(bob)]
	pose.Face = facePrompt
	pose.Arms = armsTyping
	pose.Laptop = true
	if frame%4 == 1 || frame%4 == 2 {
		pose.Face = faceBars
	}
	drawCodexPetBase(img, ox, oy, pose)
	if frame%4 == 0 {
		setPixel(img, ox+23, oy+22+pose.Y, G)
	} else if frame%4 == 2 {
		setPixel(img, ox+24, oy+21+pose.Y, G)
	}
}

func drawCodexVictory(img *image.RGBA, ox, oy, frame int) {
	pose := defaultPose()
	pose.Face = faceHappy
	pose.Arms = armsUp

	switch {
	case frame < 4:
		pose.Y = []int{2, 2, 1, 0}[frame]
		pose.SquashX = 1
		pose.SquashY = -1
	case frame < 9:
		pose.Y = []int{0, -2, -4, -6, -7}[frame-4]
		pose.SquashY = 1
	case frame < 12:
		pose.X = []int{0, 1, 0}[frame-9]
		pose.Y = -7
	case frame < 16:
		pose.Y = []int{-6, -4, -2, 0}[frame-12]
	default:
		pose.Y = []int{2, 0, -1, 0}[frame-16]
		if frame > 17 {
			pose.Arms = armsIdle
		}
	}

	drawCodexPetBase(img, ox, oy, pose)
	if frame >= 6 && frame < 18 {
		drawSparkle(img, ox+4, oy+9+pose.Y, Y)
		drawSparkle(img, ox+28, oy+10+pose.Y, G)
	}
}

func drawCodexHurt(img *image.RGBA, ox, oy, frame int) {
	pose := defaultPose()
	pose.Face = faceX
	pose.Arms = armsHurt

	if frame < 3 {
		pose.Y = 2
		pose.SquashX = -1
		pose.SquashY = -1
	} else if frame < 9 {
		pose.X = -[]int{2, 5, 7, 8, 7, 5}[frame-3]
		pose.Y = []int{0, -1, -1, 0, 1, 0}[frame-3]
	} else {
		idx := frame - 9
		pose.X = -[]int{4, 3, 2, 1, 0, 0, 0}[idx]
		pose.Y = []int{-1, 0, 1, 0, -1, 0, 0}[idx]
		if frame > 13 {
			pose.Face = faceWorry
			pose.Arms = armsIdle
		}
	}

	drawCodexPetBase(img, ox, oy, pose)
	if frame < 10 {
		drawSparkle(img, ox+28+pose.X/2, oy+8+pose.Y, Y)
		drawSparkle(img, ox+25+pose.X/2, oy+5+pose.Y, W)
	}
}

func drawCodexThinking(img *image.RGBA, ox, oy, frame int) {
	sway := []int{0, 0, 1, 1, 1, 1, 0, 0, 0, 0, -1, -1, -1, -1, 0, 0}
	bob := []int{0, 0, 0, 0, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, 0, 0}
	pose := defaultPose()
	pose.X = sway[frame%len(sway)]
	pose.Y = bob[frame%len(bob)]
	if frame%8 < 4 {
		pose.Face = faceWorry
	} else {
		pose.Face = faceBars
	}

	drawCodexPetBase(img, ox, oy, pose)
	drawThoughtDots(img, ox, oy, frame)
}

func drawCodexWalk(img *image.RGBA, ox, oy, frame int) {
	bob := []int{0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0}
	pose := defaultPose()
	pose.Y = bob[frame%len(bob)]
	pose.Legs = frame % 8
	drawCodexPetBase(img, ox, oy, pose)
}

func drawCodexVictoryPose(img *image.RGBA, ox, oy, frame int) {
	pose := defaultPose()
	pose.Face = faceHappy
	pose.Arms = armsProud

	switch {
	case frame < 4:
		pose.Y = []int{0, 1, 2, 3}[frame]
		pose.SquashX = 1
		pose.SquashY = -1
	case frame < 8:
		pose.Y = []int{1, -1, -3, -4}[frame-4]
		pose.SquashY = 1
	case frame < 14:
		pose.Y = -4 + []int{0, 1, 0, -1, 0, 1}[frame-8]
	case frame < 18:
		pose.Y = []int{-3, -2, -1, 0}[frame-14]
		if frame >= 16 {
			pose.Arms = armsUp
		}
	default:
		pose.Y = []int{1, 0}[frame-18]
		pose.Arms = armsIdle
	}

	drawCodexPetBase(img, ox, oy, pose)
	if frame >= 6 && frame < 16 {
		drawSparkle(img, ox+4, oy+7+pose.Y, Y)
		drawSparkle(img, ox+27, oy+7+pose.Y, G)
		drawSparkle(img, ox+16, oy+3+pose.Y, W)
	}
}

func drawCodexPetBase(img *image.RGBA, ox, oy int, pose petPose) {
	x := ox + pose.X
	y := oy + pose.Y

	drawPetLegs(img, x, y, pose.Legs)
	drawPetArms(img, x, y, pose.Arms)
	drawPetBody(img, x, y, pose)
	drawPetHead(img, x, y, pose)
	if pose.Laptop {
		drawPetLaptop(img, x, y)
	}
}

func drawPetHead(img *image.RGBA, x, y int, pose petPose) {
	sx := pose.SquashX
	sy := pose.SquashY / 2

	drawShadedRoundedRect(img, x+6-sx, y+6+sy, 8+sx, 7, S)
	drawShadedRoundedRect(img, x+12, y+5+sy, 8, 7, H)
	drawShadedRoundedRect(img, x+18, y+6+sy, 8+sx, 7, P)
	drawShadedRoundedRect(img, x+4-sx, y+9+sy, 24+2*sx, 9, P)

	drawRoundedRect(img, x+7-sx, y+8+sy, 18+2*sx, 9, O)
	drawRoundedRect(img, x+8-sx, y+9+sy, 16+2*sx, 7, panelFill)
	drawHorizontalLine(img, x+10-sx, x+21+sx, y+9+sy, panelHi)
	drawFace(img, x+sx, y+sy, pose.Face)
}

func drawPetBody(img *image.RGBA, x, y int, pose petPose) {
	w := clamp(12+pose.SquashX*2, 9, 16)
	h := clamp(8-pose.SquashY/2, 6, 10)
	bodyX := x + 16 - w/2
	bodyY := y + 18 + pose.SquashY/2

	drawShadedRoundedRect(img, bodyX, bodyY, w, h, P)
	drawTinyPrompt(img, x+13, bodyY+3)
}

func drawPetLegs(img *image.RGBA, x, y, phase int) {
	leftLift, rightLift := 0, 0
	leftSlide, rightSlide := 0, 0
	switch phase % 8 {
	case 1, 2:
		leftLift = 1
		leftSlide = -1
	case 3, 4:
		rightLift = 1
		rightSlide = 1
	case 5, 6:
		leftLift = 1
		leftSlide = 1
	default:
		rightLift = 0
	}

	drawShadedRoundedRect(img, x+10+leftSlide, y+24-leftLift, 4, 5, S)
	drawShadedRoundedRect(img, x+18+rightSlide, y+24-rightLift, 4, 5, P)
}

func drawPetArms(img *image.RGBA, x, y int, pose armPose) {
	switch pose {
	case armsUp:
		drawShadedRoundedRect(img, x+5, y+12, 4, 9, S)
		drawShadedRoundedRect(img, x+23, y+12, 4, 9, P)
		drawShadedRoundedRect(img, x+4, y+10, 5, 4, S)
		drawShadedRoundedRect(img, x+23, y+10, 5, 4, H)
	case armsCast:
		drawShadedRoundedRect(img, x+5, y+11, 4, 8, S)
		drawShadedRoundedRect(img, x+23, y+11, 4, 8, H)
		setPixel(img, x+5, y+10, G)
		setPixel(img, x+26, y+10, G)
	case armsPunch:
		drawShadedRoundedRect(img, x+5, y+18, 4, 5, S)
		drawShadedRoundedRect(img, x+23, y+15, 8, 4, P)
		drawShadedRoundedRect(img, x+29, y+14, 4, 5, H)
	case armsTyping:
		drawShadedRoundedRect(img, x+6, y+20, 5, 3, S)
		drawShadedRoundedRect(img, x+21, y+20, 5, 3, H)
	case armsHurt:
		drawShadedRoundedRect(img, x+4, y+20, 4, 4, S)
		drawShadedRoundedRect(img, x+24, y+19, 4, 4, H)
	case armsProud:
		drawShadedRoundedRect(img, x+5, y+9, 4, 11, S)
		drawShadedRoundedRect(img, x+23, y+9, 4, 11, P)
		drawShadedRoundedRect(img, x+4, y+7, 5, 4, S)
		drawShadedRoundedRect(img, x+23, y+7, 5, 4, H)
	default:
		drawShadedRoundedRect(img, x+5, y+18, 4, 5, S)
		drawShadedRoundedRect(img, x+23, y+18, 4, 5, H)
	}
}

func drawFace(img *image.RGBA, x, y int, face petFace) {
	switch face {
	case faceBlink:
		drawHorizontalLine(img, x+10, x+13, y+13, G)
		drawHorizontalLine(img, x+18, x+21, y+13, G)
	case faceHappy:
		setPixel(img, x+10, y+13, G)
		setPixel(img, x+11, y+12, G)
		setPixel(img, x+12, y+12, G)
		setPixel(img, x+13, y+13, G)
		setPixel(img, x+18, y+13, G)
		setPixel(img, x+19, y+12, G)
		setPixel(img, x+20, y+12, G)
		setPixel(img, x+21, y+13, G)
	case faceBars:
		drawRect(img, x+11, y+11, 2, 4, G)
		drawRect(img, x+20, y+11, 2, 4, G)
	case faceX:
		drawMiniX(img, x+10, y+11)
		drawMiniX(img, x+18, y+11)
	case faceWorry:
		drawHorizontalLine(img, x+10, x+13, y+13, G)
		drawHorizontalLine(img, x+19, x+22, y+12, G)
		setPixel(img, x+18, y+13, G)
		setPixel(img, x+22, y+13, G)
	case faceSquint:
		setPixel(img, x+10, y+11, G)
		setPixel(img, x+11, y+12, G)
		setPixel(img, x+10, y+13, G)
		setPixel(img, x+21, y+11, G)
		setPixel(img, x+20, y+12, G)
		setPixel(img, x+21, y+13, G)
	default:
		drawPromptGlyph(img, x+10, y+11)
		drawHorizontalLine(img, x+18, x+21, y+14, G)
	}
}

func drawPetLaptop(img *image.RGBA, x, y int) {
	drawRoundedRect(img, x+9, y+22, 14, 7, O)
	drawRoundedRect(img, x+10, y+23, 12, 5, M)
	drawPromptGlyph(img, x+14, y+24)
	drawHorizontalLine(img, x+8, x+24, y+28, O)
}

func drawCodexPetSmear(img *image.RGBA, ox, oy int) {
	drawShadedRoundedRect(img, ox+3, oy+9, 27, 9, P)
	drawRoundedRect(img, ox+9, oy+11, 17, 6, O)
	drawRoundedRect(img, ox+10, oy+12, 15, 4, panelFill)
	drawHorizontalLine(img, ox+13, ox+16, oy+14, G)
	drawHorizontalLine(img, ox+20, ox+23, oy+14, G)
	drawShadedRoundedRect(img, ox+24, oy+15, 8, 4, H)
	drawShadedRoundedRect(img, ox+11, oy+20, 10, 6, P)
}

func drawThoughtDots(img *image.RGBA, ox, oy, frame int) {
	dot := frame % 12
	setPixel(img, ox+23, oy+7, W)
	if dot > 3 {
		drawRect(img, ox+25, oy+5, 2, 2, W)
	}
	if dot > 7 {
		drawRoundedRect(img, ox+27, oy+2, 4, 3, W)
		setPixel(img, ox+28, oy+3, G)
	}
}

func drawTinyPrompt(img *image.RGBA, x, y int) {
	setPixel(img, x, y, G)
	setPixel(img, x+1, y+1, G)
	setPixel(img, x, y+2, G)
	drawHorizontalLine(img, x+4, x+6, y+2, G)
}

func drawPromptGlyph(img *image.RGBA, x, y int) {
	setPixel(img, x, y, G)
	setPixel(img, x+1, y+1, G)
	setPixel(img, x+2, y+2, G)
	setPixel(img, x+1, y+3, G)
	setPixel(img, x, y+4, G)
}

func drawMiniX(img *image.RGBA, x, y int) {
	setPixel(img, x, y, G)
	setPixel(img, x+2, y, G)
	setPixel(img, x+1, y+1, G)
	setPixel(img, x, y+3, G)
	setPixel(img, x+2, y+3, G)
}

func drawImpactBurst(img *image.RGBA, cx, cy int) {
	setPixel(img, cx, cy, W)
	setPixel(img, cx+1, cy, W)
	setPixel(img, cx-1, cy, W)
	setPixel(img, cx, cy-1, W)
	setPixel(img, cx, cy+1, W)
	setPixel(img, cx+3, cy-2, Y)
	setPixel(img, cx+3, cy+2, Y)
	setPixel(img, cx-2, cy-2, Y)
	setPixel(img, cx-2, cy+2, Y)
	setPixel(img, cx+4, cy, G)
	setPixel(img, cx, cy-3, G)
	setPixel(img, cx, cy+3, G)
}

func drawSparkle(img *image.RGBA, x, y int, c C) {
	setPixel(img, x, y, W)
	setPixel(img, x+1, y, c)
	setPixel(img, x-1, y, c)
	setPixel(img, x, y+1, c)
	setPixel(img, x, y-1, c)
}

func drawShadedRoundedRect(img *image.RGBA, x, y, w, h int, fill C) {
	if w <= 0 || h <= 0 {
		return
	}
	drawRoundedRect(img, x, y, w, h, O)
	if w <= 2 || h <= 2 {
		return
	}
	drawRoundedRect(img, x+1, y+1, w-2, h-2, fill)
	drawHorizontalLine(img, x+2, x+w-3, y+1, H)
	drawHorizontalLine(img, x+2, x+w-3, y+h-2, S)
	for yy := y + 2; yy <= y+h-3; yy++ {
		setPixel(img, x+1, yy, S)
		setPixel(img, x+w-2, yy, H)
	}
}

func drawRoundedRect(img *image.RGBA, x, y, w, h int, c C) {
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			if (xx == 0 || xx == w-1) && (yy == 0 || yy == h-1) {
				continue
			}
			setPixel(img, x+xx, y+yy, c)
		}
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, c C) {
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			setPixel(img, x+xx, y+yy, c)
		}
	}
}

func drawHorizontalLine(img *image.RGBA, x1, x2, y int, c C) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		setPixel(img, x, y, c)
	}
}

func setPixel(img *image.RGBA, x, y int, c C) {
	img.Set(x, y, c)
}

func simpleSin(deg int) int {
	sins := []int{0, 50, 87, 100, 87, 50, 0, -50, -87, -100, -87, -50}
	return sins[((deg/30)%12+12)%12]
}

func simpleCos(deg int) int {
	return simpleSin(deg + 90)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
