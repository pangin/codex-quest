package main

import rl "github.com/gen2brain/raylib-go/raylib"

func (r *Renderer) drawCodex(state *AnimationState) {
	// Scaled dimensions
	scaledW := float32(spriteFrameWidth * codexScale)
	scaledH := float32(spriteFrameHeight * codexScale)

	// Position Codex in center of scene
	x := float32(screenWidth/2) - scaledW/2
	y := float32(160) - scaledH + 10 // Feet on floor

	if r.hasSprites {
		// Calculate source rectangle from sprite sheet
		frameX := float32(state.Frame * spriteFrameWidth)
		frameY := float32(int(state.CurrentAnim) * spriteFrameHeight)

		sourceRec := rl.Rectangle{
			X:      frameX,
			Y:      frameY,
			Width:  spriteFrameWidth,
			Height: spriteFrameHeight,
		}

		destRec := rl.Rectangle{
			X:      x,
			Y:      y,
			Width:  scaledW,
			Height: scaledH,
		}

		rl.DrawTexturePro(r.spriteSheet, sourceRec, destRec, rl.Vector2{}, 0, rl.White)
	} else {
		// Fallback placeholder
		r.drawPlaceholderCodex(int(x), int(y), state)
	}
}

// getHeadOffset returns the X,Y offset of Codex's head for the current animation frame.
// These offsets match the sprite generator's Pet-style base pose.
func getHeadOffset(state *AnimationState) (float32, float32) {
	f := state.Frame

	switch state.CurrentAnim {
	case AnimIdle:
		bob := []int{0, 0, 0, -1, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, 0, 0}
		return 0, float32(bob[f%len(bob)])

	case AnimEnter:
		if f < 8 {
			return 0, -100 // Off screen
		} else if f < 15 {
			settleY := []int{3, 2, 1, 0, -1, 0, 0}
			return 0, float32(settleY[f-8])
		} else {
			bounce := []int{-2, -1, 0, 0, 0}
			return 0, float32(bounce[f-15])
		}

	case AnimCasting:
		if f < 5 {
			windup := []int{2, 2, 1, 0, -1}
			return 0, float32(windup[f])
		} else if f < 13 {
			floatY := []int{-2, -3, -3, -2, -2, -3, -2, -2}
			return 0, float32(floatY[f-5])
		} else {
			settleY := []int{-1, 0, 0}
			idx := f - 13
			if idx >= len(settleY) {
				idx = len(settleY) - 1
			}
			return 0, float32(settleY[idx])
		}

	case AnimAttack:
		if f < 3 {
			y := []int{1, 2, 2}
			return float32(-f), float32(y[f])
		} else if f < 5 {
			return -2, 3
		} else if f < 7 {
			y := []int{2, 1}
			return 0, float32(y[f-5])
		} else if f == 7 {
			return 0, 0
		} else if f < 10 {
			return 2, 1
		} else if f < 14 {
			xOff := []int{1, 1, 0, 0}
			yOff := []int{-1, 0, 1, 0}
			return float32(xOff[f-10]), float32(yOff[f-10])
		} else {
			bounce := []int{-1, 0}
			idx := f - 14
			if idx >= len(bounce) {
				idx = len(bounce) - 1
			}
			return 0, float32(bounce[idx])
		}

	case AnimWriting:
		bob := []int{0, -1, -1, 0, 0, -1, -1, 0, 0, -1, -1, 0, 0, -1, 0, 0}
		return 0, float32(bob[f%len(bob)])

	case AnimVictory:
		if f < 4 {
			y := []int{2, 2, 1, 0}
			return 0, float32(y[f])
		} else if f < 9 {
			y := []int{0, -2, -4, -6, -7}
			return 0, float32(y[f-4])
		} else if f < 12 {
			wiggle := []int{0, 1, 0}
			return float32(wiggle[f-9]), -7
		} else if f < 16 {
			y := []int{-6, -4, -2, 0}
			return 0, float32(y[f-12])
		} else {
			bounceY := []int{2, 0, -1, 0}
			idx := f - 16
			if idx >= len(bounceY) {
				idx = len(bounceY) - 1
			}
			return 0, float32(bounceY[idx])
		}

	case AnimHurt:
		if f < 3 {
			return 0, 2
		} else if f < 9 {
			knockX := []int{2, 5, 7, 8, 7, 5}
			y := []int{0, -1, -1, 0, 1, 0}
			return float32(-knockX[f-3]), float32(y[f-3])
		} else {
			recoverX := []int{4, 3, 2, 1, 0, 0, 0}
			bounceY := []int{-1, 0, 1, 0, -1, 0, 0}
			idx := f - 9
			if idx >= len(recoverX) {
				idx = len(recoverX) - 1
			}
			return float32(-recoverX[idx]), float32(bounceY[idx])
		}

	case AnimThinking:
		swayCurve := []int{0, 0, 1, 1, 1, 1, 0, 0, 0, 0, -1, -1, -1, -1, 0, 0}
		bobCurve := []int{0, 0, 0, 0, -1, -1, -1, -1, -1, 0, 0, 0, 0, 0, 0, 0}
		return float32(swayCurve[f%len(swayCurve)]), float32(bobCurve[f%len(bobCurve)])

	case AnimWalk:
		bobCurve := []int{0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0}
		return 0, float32(bobCurve[f%len(bobCurve)])

	case AnimVictoryPose:
		if f < 4 {
			y := []int{0, 1, 2, 3}
			return 0, float32(y[f])
		} else if f < 8 {
			riseY := []int{1, -1, -3, -4}
			return 0, float32(riseY[f-4])
		} else if f < 14 {
			bob := []int{0, 1, 0, -1, 0, 1}
			return 0, float32(-4 + bob[f-8])
		} else if f < 18 {
			settleY := []int{-3, -2, -1, 0}
			return 0, float32(settleY[f-14])
		} else {
			bounceY := []int{1, 0}
			idx := f - 18
			if idx >= len(bounceY) {
				idx = len(bounceY) - 1
			}
			return 0, float32(bounceY[idx])
		}
	}

	return 0, 0
}

func (r *Renderer) drawHat(state *AnimationState) {
	// Use preview hat when modal picker is open
	hatIdx := r.GetPreviewHat()
	if hatIdx < 0 || hatIdx >= len(r.hats) {
		return
	}

	hat := r.hats[hatIdx]
	hatName := r.hatNames[hatIdx]

	// Get animation-specific offset (in sprite pixels, before scaling)
	headOffX, headOffY := getHeadOffset(state)

	// Don't draw if off-screen (e.g., during Enter sparkles phase)
	if headOffY < -50 {
		return
	}

	// Codex's base position in SCREEN coords (same as drawCodex)
	scaledW := float32(spriteFrameWidth * codexScale)
	scaledH := float32(spriteFrameHeight * codexScale)
	codexX := float32(screenWidth/2) - scaledW/2
	codexY := float32(160) - scaledH + 10

	// Hat dimensions (scale with Codex)
	hatW := float32(hat.Width) * float32(codexScale)
	hatH := float32(hat.Height) * float32(codexScale)

	var hatX, hatY float32

	// Special positioning for headphones - wrap around head at ear level
	if hatName == "headphones" {
		// Headphones: band on top, cups at ear level (sprite y ~14-17)
		// Position so the top of headphones aligns with top of head
		// Scale wider to wrap around head properly
		hatW = hatW * 1.4
		spriteHeadY := float32(6) // Top of new rounded head
		hatX = codexX + scaledW/2 - hatW/2 + headOffX*float32(codexScale)
		hatY = codexY + (spriteHeadY+headOffY)*float32(codexScale)
	} else if hatName == "zeus" {
		// Zeus hair is a wig that frames the face with beard below
		// Scale wider to wrap around head properly
		hatW = hatW * 1.4
		spriteHeadY := float32(4) // Slightly above the helmet silhouette
		hatX = codexX + scaledW/2 - hatW/2 + headOffX*float32(codexScale)
		hatY = codexY + (spriteHeadY+headOffY)*float32(codexScale)
	} else {
		// Standard hat positioning - sits on the Pet helmet top.
		spriteHeadY := float32(6)

		// Convert to screen coords:
		// codexY is the top-left of the 32x32 sprite frame (scaled)
		// Add spriteHeadY * scale to get head position
		// Add animation offset * scale
		hatX = codexX + scaledW/2 - hatW/2 + headOffX*float32(codexScale)
		hatY = codexY + (spriteHeadY+headOffY)*float32(codexScale) - hatH + 2*float32(codexScale)
	}

	sourceRec := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(hat.Width),
		Height: float32(hat.Height),
	}

	destRec := rl.Rectangle{
		X:      hatX,
		Y:      hatY,
		Width:  hatW,
		Height: hatH,
	}

	rl.DrawTexturePro(hat, sourceRec, destRec, rl.Vector2{}, 0, rl.White)
}

func (r *Renderer) drawFace(state *AnimationState) {
	// Use preview face when modal picker is open
	faceIdx := r.GetPreviewFace()
	if faceIdx < 0 || faceIdx >= len(r.faces) {
		return
	}

	face := r.faces[faceIdx]
	faceName := r.faceNames[faceIdx]

	// Get animation-specific offset
	headOffX, headOffY := getHeadOffset(state)

	// Don't draw if off-screen
	if headOffY < -50 {
		return
	}

	// Codex's base position
	scaledW := float32(spriteFrameWidth * codexScale)
	scaledH := float32(spriteFrameHeight * codexScale)
	codexX := float32(screenWidth/2) - scaledW/2
	codexY := float32(160) - scaledH + 10

	// Face accessory dimensions
	faceW := float32(face.Width) * float32(codexScale)
	faceH := float32(face.Height) * float32(codexScale)

	// Position depends on accessory type
	// New Pet face panel is at ~y=8-16, eyes around y=12-13, body at y=18-26.
	var spriteY float32
	var spriteXOffset float32 = 0 // offset from center
	var centerVertically bool = true
	switch faceName {
	case "dealwithit":
		// Goes across the cyan eye glyphs.
		spriteY = 12
	case "monocle":
		// Monocle on RIGHT eye within the face panel.
		spriteY = 9
		spriteXOffset = 3
		centerVertically = false // position from top of sprite
	case "mustache":
		// Lower edge of the face panel.
		spriteY = 16
	case "pipe":
		// Pipe stem comes from the lower-right face panel.
		spriteY = 13
		spriteXOffset = 3
		centerVertically = false
	case "eyepatch":
		// Eyepatch over left eye in the face panel.
		spriteY = 9
		spriteXOffset = -3
		centerVertically = false
	case "wizardbeard":
		// Beard hangs from the helmet/face-panel bottom.
		spriteY = 15
		centerVertically = false
	case "bandana":
		// Bandana sits across the helmet forehead.
		spriteY = 7
		centerVertically = false
	case "borat":
		// Body costume: straps at body top, pouch near feet.
		spriteY = 20
	default:
		spriteY = 13
	}

	faceX := codexX + scaledW/2 - faceW/2 + (headOffX+spriteXOffset)*float32(codexScale)
	faceY := codexY + (spriteY+headOffY)*float32(codexScale)
	if centerVertically {
		faceY -= faceH / 2
	}

	sourceRec := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(face.Width),
		Height: float32(face.Height),
	}

	destRec := rl.Rectangle{
		X:      faceX,
		Y:      faceY,
		Width:  faceW,
		Height: faceH,
	}

	rl.DrawTexturePro(face, sourceRec, destRec, rl.Vector2{}, 0, rl.White)
}

func (r *Renderer) drawPlaceholderCodex(x, y int, state *AnimationState) {
	// Simple placeholder when no sprites loaded
	color := rl.Color{R: 93, G: 141, B: 255, A: 255}
	panel := rl.Color{R: 11, G: 22, B: 56, A: 255}
	cyan := rl.Color{R: 114, G: 246, B: 255, A: 255}

	bobOffset := 0
	if state.CurrentAnim == AnimIdle {
		bobOffset = int(state.Frame/10) % 2
	}

	// Body
	rl.DrawRectangle(int32(x+22), int32(y+36), 20, 18, color)
	// Head
	rl.DrawCircle(int32(x+32), int32(y+22+bobOffset), 22, color)
	// Face panel
	rl.DrawRectangleRounded(rl.Rectangle{X: float32(x + 16), Y: float32(y + 18 + bobOffset), Width: 32, Height: 16}, 0.25, 4, panel)
	rl.DrawText(">_", int32(x+23), int32(y+20+bobOffset), 12, cyan)
}
