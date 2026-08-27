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

	if !renderer.Init(1600, 1000) {
		return
	}

	renderer.SetFog(100.0, 600.0, 0.25, 0.25, 0.25, 1.0)

	// --- Szene aufbauen ---
	boxMesh := solidBox(100, 80, 60) // CPU-Geometrie; Upload in den GPU-Batch pro Frame
	pyrMesh := solidPyramid(90, 120)
	gridMesh := solidGrid(600, 24)

	bodies := make([]Body, 0, 6)

	grid := NewBody(gridMesh, 0, 0, 0, BodyConfig{Color: "#777774", LineWidth: 1.0})
	bodies = append(bodies, grid)

	box1 := NewBody(boxMesh, 150, 0, 50, BodyConfig{Color: "#ff0000", LineWidth: 2.0})
	bodies = append(bodies, box1)

	box2 := NewBody(boxMesh, 0, 0, 100, BodyConfig{Color: "#00ffff", LineWidth: 2.0})
	bodies = append(bodies, box2)

	box3 := NewBody(boxMesh, -150, 0, -100, BodyConfig{Color: "#ff0000", LineWidth: 2.0})
	bodies = append(bodies, box3)

	box4 := NewBody(boxMesh, -200, 0, 30, BodyConfig{Color: "#00ffff", LineWidth: 2.0, RotY: float32(math.Pi / 2)})
	bodies = append(bodies, box4)

	pyr1 := NewBody(pyrMesh, 100, 0, -100, bodyConfigDefault)
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
		l := NewLine(apex, end, "#ff8800", 1)
		lines = append(lines, l)
	}

	timeAccum := float32(0)
	coneRotY := float32(0)
	frameCount := 0
	fpsLast := float64(0)

	/*---------------------------------
	Render-Schleife (in der Library)
	---------------------------------*/
	renderer.StartAnimation(func() {
		timeAccum += 0.02
		frameCount++
		now := glfw.GetTime()
		if now-fpsLast >= 2.0 {
			fps := float64(frameCount) / (now - fpsLast)
			fmt.Printf("FPS: %.1f\n", fps)
			frameCount = 0
			fpsLast = now
		}

		renderer.Background(40, 40, 40)

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
		proj := mat4x4Perspective(1.2, renderer.Aspect(), 0.1, 1000.0)
		renderer.SetProjection(&proj)

		bodyCount := len(bodies)
		for i := 0; i < bodyCount; i++ {
			bodies[i].Draw(&view)
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
				worldVerts[j] = b.Solid.Vertices[j].Transform(&rot).Add(b.Pos)
			}
			boxPlanes[i] = b.GetFacePlanes(worldVerts)
		}

		for i := 0; i < lineCount; i++ {
			rotatedEnd := lines[i].P2.RotateAround(lines[i].P1, &coneRot)
			endpoint := rotatedEnd
			maxDist := rotatedEnd.Sub(lines[i].P1).SquaredLength()

			for j := 0; j < bodyCount; j++ {
				for k := 0; k < boxPlanes[j].Count; k++ {
					var hit Vec3
					if boxPlanes[j].Data[k].IntersectLine(lines[i].P1, rotatedEnd, &hit) {
						dist := hit.Sub(lines[i].P1).SquaredLength()
						if dist < maxDist {
							endpoint = hit
							maxDist = dist
						}
					}
				}
			}

			lines[i].Draw(&view, &endpoint)
			renderer.SetStrokeColorHex("#ff0000")
			renderer.SetPointSize(5)
			renderer.Point(endpoint.X, endpoint.Y, endpoint.Z)
		}
	})

	// Aufräumen übernimmt renderer.Close() (GLFW-Terminate gibt alle
	// GL-Objekte frei) – analog zum Vehicle-Beispiel.
}
