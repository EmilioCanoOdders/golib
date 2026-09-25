package main

import "golib"

const (
	modelFrameWidth  = 128
	modelFrameHeight = 192
	modelFootX       = 64
	modelFootY       = 160
	modelRenderZoom  = 32
)

type worldSprite struct {
	sprite        *golib.Sprite
	shadow        *golib.Sprite
	visibleAcross float32
	minPixels     float32
}

func newWorldSprite(
	name, shadowName string, across, minimum float32,
) worldSprite {
	return worldSprite{
		sprite: golib.NewSpriteSheet(
			name, modelFrameWidth, modelFrameHeight,
		),
		shadow: golib.NewSpriteSheet(
			shadowName, modelFrameWidth, modelFrameHeight,
		),
		visibleAcross: across, minPixels: minimum,
	}
}

var (
	coreWorkerModel = newWorldSprite(
		"sprites/worker-core.png",
		"sprites/shadows/worker-core.png", 4, 6,
	)
	carrierModel = newWorldSprite(
		"sprites/worker-carrier.png",
		"sprites/shadows/worker-carrier.png", 5, 6,
	)
	trooperModel = newWorldSprite(
		"sprites/worker-trooper.png",
		"sprites/shadows/worker-trooper.png", 5, 6,
	)
	mechanicModel = newWorldSprite(
		"sprites/worker-mechanic.png",
		"sprites/shadows/worker-mechanic.png", 5, 6,
	)
	rivalScoutModel = newWorldSprite(
		"sprites/rival-scout.png",
		"sprites/shadows/rival-scout.png", 3, 5,
	)
	rivalCrawlerModel = newWorldSprite(
		"sprites/rival-crawler.png",
		"sprites/shadows/rival-crawler.png", 8, 9,
	)
	rivalRaiderModel = newWorldSprite(
		"sprites/rival-raider.png",
		"sprites/shadows/rival-raider.png", 4, 5,
	)
	rivalArtilleryModel = newWorldSprite(
		"sprites/rival-artillery.png",
		"sprites/shadows/rival-artillery.png", 12, 8,
	)
)

func robotModel(kind RobotKind) worldSprite {
	switch kind {
	case RobotBuilder:
		return coreWorkerModel
	case RobotWorker:
		return carrierModel
	case RobotCombat:
		return trooperModel
	case RobotRepair:
		return mechanicModel
	default:
		return coreWorkerModel
	}
}

func modelScreenPosition(camera *golib.Camera, position golib.Vector2) golib.Vector2 {
	return camera.ToScreen(position)
}

func (m worldSprite) iconScale(zoom float32) float32 {
	return max(1, m.minPixels/(m.visibleAcross*unitW*zoom))
}

func (m worldSprite) draw(
	screen *golib.Screen, camera *golib.Camera,
	position golib.Vector2, facing uint8, zoom float32,
) {
	at := modelScreenPosition(camera, position)
	scale := m.iconScale(zoom)
	// DrawSprite rounds before applying a camera: draw at screen pixels so
	// one world-pixel step does not become a zoom-sized jump.
	screen.SetCamera(nil)
	screen.DrawSprite(m.sprite, int(facing%8), at.X, at.Y,
		golib.DrawOptions{
			OriginX: modelFootX,
			OriginY: modelFootY,
			Scale:   scale * zoom / modelRenderZoom,
		})
	screen.SetCamera(camera)
}

func (m worldSprite) drawShadow(
	screen *golib.Screen, camera *golib.Camera,
	position golib.Vector2, facing uint8, zoom float32,
) {
	at := modelScreenPosition(camera, position)
	scale := m.iconScale(zoom) * zoom / modelRenderZoom
	screen.SetCamera(nil)
	screen.DrawSprite(m.shadow, int(facing%8), at.X, at.Y,
		golib.DrawOptions{
			OriginX: modelFootX,
			OriginY: modelFootY,
			Scale:   scale,
			Tint:    unitShadowTint,
		})
	screen.SetCamera(camera)
}

func (m worldSprite) drawIcon(
	screen *golib.Screen, position golib.Vector2,
	facing uint8, tint golib.Color,
) {
	screen.DrawSprite(m.sprite, int(facing%8), position.X, position.Y,
		golib.DrawOptions{
			OriginX: modelFootX,
			OriginY: 147,
			Scale:   0.5,
			Tint:    tint,
		})
}
