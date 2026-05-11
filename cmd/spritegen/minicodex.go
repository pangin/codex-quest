package main

import (
	"image"
	"image/png"
	"os"
)

// Mini Codex sprite sheet generator.
// Mini Codex is 16x16 pixels with Spawn, Idle, Walk, and Poof animations.

const (
	miniFrameWidth  = 16
	miniFrameHeight = 16
	miniNumAnims    = 4
	miniMaxFrames   = 12
)

var miniFrameCounts = []int{8, 8, 8, 6}

func generateMiniCodex() {
	width := miniFrameWidth * miniMaxFrames
	height := miniFrameHeight * miniNumAnims
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, X)
		}
	}

	for anim := 0; anim < miniNumAnims; anim++ {
		for frame := 0; frame < miniFrameCounts[anim]; frame++ {
			drawMiniFrame(img, anim, frame)
		}
	}

	os.MkdirAll("assets/codex", 0755)
	f, _ := os.Create("assets/codex/mini_spritesheet.png")
	defer f.Close()
	png.Encode(f, img)
}

func drawMiniFrame(img *image.RGBA, anim, frame int) {
	offsetX := frame * miniFrameWidth
	offsetY := anim * miniFrameHeight

	switch anim {
	case 0:
		drawMiniSpawn(img, offsetX, offsetY, frame)
	case 1:
		drawMiniIdle(img, offsetX, offsetY, frame)
	case 2:
		drawMiniWalk(img, offsetX, offsetY, frame)
	case 3:
		drawMiniPoof(img, offsetX, offsetY, frame)
	}
}

func drawMiniSpawn(img *image.RGBA, ox, oy, frame int) {
	switch frame {
	case 0:
		drawSparkle(img, ox+8, oy+14, Y)
	case 1:
		drawRect(img, ox+6, oy+11, 4, 2, P)
		drawSparkle(img, ox+8, oy+13, G)
	case 2:
		drawMiniPet(img, ox, oy, 3, faceBars, 0)
		drawSparkle(img, ox+4, oy+12, Y)
	case 3:
		drawMiniPet(img, ox, oy, -3, faceHappy, 0)
		drawSparkle(img, ox+2, oy+5, G)
		drawSparkle(img, ox+13, oy+4, Y)
	case 4:
		drawMiniPet(img, ox, oy, -2, faceHappy, 0)
	case 5:
		drawMiniPet(img, ox, oy, 0, facePrompt, 0)
	case 6:
		drawMiniPet(img, ox, oy, 1, faceBlink, 0)
	case 7:
		drawMiniPet(img, ox, oy, 0, faceHappy, 0)
		drawSparkle(img, ox+2, oy+12, Y)
		drawSparkle(img, ox+13, oy+12, G)
	}
}

func drawMiniIdle(img *image.RGBA, ox, oy, frame int) {
	bounce := []int{0, 0, -1, -1, 0, 0, 1, 0}
	face := facePrompt
	if frame == 5 || frame == 6 {
		face = faceBlink
	}
	if frame == 7 {
		face = faceHappy
	}
	drawMiniPet(img, ox, oy, bounce[frame%len(bounce)], face, 0)
}

func drawMiniWalk(img *image.RGBA, ox, oy, frame int) {
	bob := []int{0, 0, 1, 1, 0, 0, 1, 0}
	drawMiniPet(img, ox, oy, bob[frame%len(bob)], facePrompt, frame)
}

func drawMiniPoof(img *image.RGBA, ox, oy, frame int) {
	switch frame {
	case 0:
		drawMiniPet(img, ox, oy, 0, faceHappy, 0)
		drawSparkle(img, ox+2, oy+4, Y)
		drawSparkle(img, ox+13, oy+4, G)
	case 1:
		drawMiniPet(img, ox, oy, 0, faceHappy, 0)
		drawSparkle(img, ox+1, oy+3, Y)
		drawSparkle(img, ox+14, oy+3, G)
	case 2:
		drawShadedRoundedRect(img, ox+5, oy+6, 6, 5, P)
		drawRect(img, ox+6, oy+7, 4, 2, panelFill)
		drawSparkle(img, ox+2, oy+5, Y)
		drawSparkle(img, ox+13, oy+5, G)
	case 3:
		drawShadedRoundedRect(img, ox+6, oy+7, 4, 3, P)
		drawSparkle(img, ox+3, oy+4, Y)
		drawSparkle(img, ox+12, oy+4, W)
		drawSparkle(img, ox+8, oy+12, G)
	case 4:
		drawRect(img, ox+7, oy+8, 2, 2, P)
		drawSparkle(img, ox+4, oy+5, Y)
		drawSparkle(img, ox+11, oy+5, G)
		drawSparkle(img, ox+2, oy+8, W)
		drawSparkle(img, ox+13, oy+8, Y)
	case 5:
		drawSparkle(img, ox+3, oy+4, Y)
		drawSparkle(img, ox+12, oy+4, W)
		drawSparkle(img, ox+7, oy+7, G)
		drawSparkle(img, ox+12, oy+12, Y)
	}
}

func drawMiniPet(img *image.RGBA, ox, oy, yOffset int, face petFace, legPhase int) {
	x := ox
	y := oy + yOffset

	// Legs and arms first so the head/body read cleanly at 16x16.
	leftLift, rightLift := 0, 0
	if legPhase%4 == 1 {
		leftLift = 1
	} else if legPhase%4 == 3 {
		rightLift = 1
	}
	drawShadedRoundedRect(img, x+5, y+11-leftLift, 2, 3, S)
	drawShadedRoundedRect(img, x+9, y+11-rightLift, 2, 3, P)
	setPixel(img, x+2, y+8, S)
	setPixel(img, x+13, y+8, H)

	drawShadedRoundedRect(img, x+5, y+9, 6, 4, P)
	drawRoundedRect(img, x+3, y+4, 10, 6, P)
	drawRoundedRect(img, x+4, y+5, 8, 4, O)
	drawRoundedRect(img, x+5, y+6, 6, 2, panelFill)

	switch face {
	case faceHappy:
		setPixel(img, x+5, y+7, G)
		setPixel(img, x+6, y+6, G)
		setPixel(img, x+9, y+6, G)
		setPixel(img, x+10, y+7, G)
	case faceBlink:
		drawHorizontalLine(img, x+5, x+6, y+7, G)
		drawHorizontalLine(img, x+9, x+10, y+7, G)
	case faceBars:
		setPixel(img, x+5, y+6, G)
		setPixel(img, x+5, y+7, G)
		setPixel(img, x+10, y+6, G)
		setPixel(img, x+10, y+7, G)
	default:
		setPixel(img, x+5, y+6, G)
		setPixel(img, x+6, y+7, G)
		drawHorizontalLine(img, x+9, x+10, y+7, G)
	}

	setPixel(img, x+7, y+10, G)
	setPixel(img, x+8, y+11, G)
}
