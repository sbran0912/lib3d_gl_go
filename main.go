package main

import (
	"fmt"
	"math"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
)

const coneLines = 20

func main() {
	runtime.LockOSThread()

	if !renderInit(1600, 1000) {
		return
	}

	renderSetFog(100.0, 600.0, 0.25, 0.25, 0.25, 1.0)

	// --- Szene aufbauen ---
	boxMesh := solidBox(100, 80, 60)     // Eine Box im GPU-Speicher
	pyrMesh := solidPyramid(90, 120)     // Eine Pyramide im GPU-Speicher
	gridMesh := solidGrid(600, 24)

	bodies := make([]Body, 0, 6)

	grid := bodyCreate(gridMesh, 0, 0, 0, BodyConfig{Color: "#777774", LineWidth: 1.0})
	bodies = append(bodies, grid)

	box1 := bodyCreate(boxMesh, 150, 0, 50, BodyConfig{Color: "#ff0000", LineWidth: 2.0})
	bodies = append(bodies, box1)

	box2 := bodyCreate(boxMesh, 0, 0, 100, BodyConfig{Color: "#00ffff", LineWidth: 2.0})
	bodies = append(bodies, box2)

	box3 := bodyCreate(boxMesh, -150, 0, -100, BodyConfig{Color: "#ff0000", LineWidth: 2.0})
	bodies = append(bodies, box3)

	box4 := bodyCreate(boxMesh, -200, 0, 30, BodyConfig{Color: "#00ffff", LineWidth: 2.0, RotY: float32(math.Pi / 2)})
	bodies = append(bodies, box4)

	pyr1 := bodyCreate(pyrMesh, 100, 0, -100, bodyConfigDefault)
	bodies = append(bodies, pyr1)

	lines := make([]*Line, 0, coneLines)

	apex := vec3(0, 0, 0)
	coneLen := float32(600)
	coneAngle := float32(math.Pi / 60)
	coneR := coneLen * float32(math.Sin(float64(coneAngle)))
	coneZ := coneLen * float32(math.Cos(float64(coneAngle)))

	for i := 0; i < coneLines; i++ {
		a := 2.0 * float32(math.Pi) * float32(i) / float32(coneLines)
		end := vec3(
			apex.X+float32(math.Cos(float64(a)))*coneR,
			apex.Y+float32(math.Sin(float64(a)))*coneR,
			apex.Z+coneZ,
		)
		l := lineCreate(apex, end, "#ff8800", 1)
		lines = append(lines, l)
	}

	timeAccum := float32(0)
	coneRotY := float32(0)
	frameCount := 0
	fpsLast := float64(0)

	// --- Render-Loop ---
	for !renderShouldClose() {
		renderFrameBegin()

		timeAccum += 0.02
		frameCount++
		now := glfw.GetTime()
		if now-fpsLast >= 2.0 {
			fps := float64(frameCount) / (now - fpsLast)
			fmt.Printf("FPS: %.1f\n", fps)
			frameCount = 0
			fpsLast = now
		}

		renderBackground(40, 40, 40)

		camAngle := timeAccum * 0.15
		camRadius := float32(math.Sqrt(float64(40.0*40.0 + 180.0*180.0)))
		camHeight := float32(140.0)
		camPos := vec3(
			float32(math.Sin(float64(camAngle)))*camRadius,
			camHeight,
			float32(math.Cos(float64(camAngle)))*camRadius,
		)
		target := vec3(0, 0, 0)
		up := vec3(0, 1, 0)

		view := mat4x4Lookat(camPos, target, up)
		proj := mat4x4Perspective(1.2, renderGetAspect(), 0.1, 1000.0)
		renderSetProjection(&proj)

		bodyCount := len(bodies)
		for i := 0; i < bodyCount; i++ {
			bodyDraw(&bodies[i], &view)
		}

		coneRotY += 0.01
		coneRot := mat4x4Rotate(0, coneRotY, 0)

		lineCount := len(lines)
		boxPlanes := make([]PlaneArray, bodyCount)
		for i := 0; i < bodyCount; i++ {
			b := &bodies[i]
			vc := b.Solid.VertexCount
			worldVerts := make([]Vec3, vc)
			rot := mat4x4Rotate(b.RotX, b.RotY, b.RotZ)
			for j := 0; j < vc; j++ {
				worldVerts[j] = vec3Add(vec3Transform(b.Solid.Vertices[j], &rot), b.Pos)
			}
			boxPlanes[i] = bodyGetFacePlanes(b, worldVerts)
		}

		for i := 0; i < lineCount; i++ {
			rotatedEnd := rotateAround(lines[i].P2, lines[i].P1, &coneRot)
			endpoint := rotatedEnd
			maxDist := vec3SquaredLength(vec3Sub(rotatedEnd, lines[i].P1))

			for j := 0; j < bodyCount; j++ {
				for k := 0; k < boxPlanes[j].Count; k++ {
					var hit Vec3
					if planeIntersectLine(&boxPlanes[j].Data[k], lines[i].P1, rotatedEnd, &hit) {
						dist := vec3SquaredLength(vec3Sub(hit, lines[i].P1))
						if dist < maxDist {
							endpoint = hit
							maxDist = dist
						}
					}
				}
			}

			lineDraw(lines[i], &view, &endpoint)
			renderStrokeColorHex("#ff0000")
			renderPointSize(5)
			renderPoint(endpoint.X, endpoint.Y, endpoint.Z)
		}

		for i := 0; i < bodyCount; i++ {
			planeArrayFree(&boxPlanes[i])
		}

		renderFrameEnd()
	}

	// --- Aufräumen ---
	for i := range bodies {
		bodyDestroy(&bodies[i])
	}

	glfw.Terminate()
}
