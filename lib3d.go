package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const epsilon = 1e-10

type Vec3 struct {
	X, Y, Z float32
}

type Vec2 struct {
	X, Y, S float32
}

type Mat4x4 struct {
	M [16]float32
}

type DrawStyle int

const (
	Stroke DrawStyle = iota
	Fill
	Both
)

type EffectMode int

const (
	EffectFlat EffectMode = iota
	EffectGradient
	EffectPulse
)

type Solid struct {
	Vertices        []Vec3
	VertexCount     int
	Edges           []int
	EdgeCount       int
	Faces           []int
	FaceCount       int
	FaceSizes       []int
	RefCount        int
	MeshBuffer      uint32
	MeshVertexCount int
}

type Plane struct {
	Normal        Vec3
	Distance      float32
	Boundary      []Vec3
	BoundaryCount int
}

func vec3(x, y, z float32) Vec3 {
	return Vec3{x, y, z}
}

func (a Vec3) Add(b Vec3) Vec3 {
	return vec3(a.X+b.X, a.Y+b.Y, a.Z+b.Z)
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return vec3(a.X-b.X, a.Y-b.Y, a.Z-b.Z)
}

func (a Vec3) Scale(s float32) Vec3 {
	return vec3(a.X*s, a.Y*s, a.Z*s)
}

func (a Vec3) SetMagnitude(m float32) Vec3 {
	l := a.Length()
	if l == 0 {
		return vec3(0, 0, 0)
	}
	return a.Scale(m / l)
}

func (a Vec3) Limit(max float32) Vec3 {
	mSq := a.SquaredLength()
	if mSq > max*max {
		return a.Scale(max / float32(math.Sqrt(float64(mSq))))
	}
	return a
}

func (a Vec3) Negate() Vec3 {
	return vec3(-a.X, -a.Y, -a.Z)
}

func (a Vec3) Dot(b Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return vec3(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) SquaredLength() float32 {
	return a.X*a.X + a.Y*a.Y + a.Z*a.Z
}

func (a Vec3) Length() float32 {
	return float32(math.Sqrt(float64(a.SquaredLength())))
}

func (a Vec3) Distance(b Vec3) float32 {
	return a.Sub(b).Length()
}

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l == 0 {
		return vec3(0, 0, 0)
	}
	return a.Scale(1 / l)
}

func (a Vec3) Lerp(b Vec3, t float32) Vec3 {
	return a.Add(b.Sub(a).Scale(t))
}

func (a Vec3) Clone() Vec3 {
	return a
}

func (a Vec3) Equals(b Vec3) bool {
	return math.Abs(float64(a.X-b.X)) < epsilon &&
		math.Abs(float64(a.Y-b.Y)) < epsilon &&
		math.Abs(float64(a.Z-b.Z)) < epsilon
}

func (v Vec3) Transform(m *Mat4x4) Vec3 {
	return vec3(
		m.M[0]*v.X+m.M[4]*v.Y+m.M[8]*v.Z+m.M[12],
		m.M[1]*v.X+m.M[5]*v.Y+m.M[9]*v.Z+m.M[13],
		m.M[2]*v.X+m.M[6]*v.Y+m.M[10]*v.Z+m.M[14],
	)
}

func mat4x4Identity() Mat4x4 {
	return Mat4x4{[16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}}
}

func mat4x4Translate(dx, dy, dz float32) Mat4x4 {
	return Mat4x4{[16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		dx, dy, dz, 1,
	}}
}

func mat4x4Rotate(ax, ay, az float32) Mat4x4 {
	rx := Mat4x4{[16]float32{
		1, 0, 0, 0,
		0, float32(math.Cos(float64(ax))), float32(math.Sin(float64(ax))), 0,
		0, -float32(math.Sin(float64(ax))), float32(math.Cos(float64(ax))), 0,
		0, 0, 0, 1,
	}}

	ry := Mat4x4{[16]float32{
		float32(math.Cos(float64(ay))), 0, -float32(math.Sin(float64(ay))), 0,
		0, 1, 0, 0,
		float32(math.Sin(float64(ay))), 0, float32(math.Cos(float64(ay))), 0,
		0, 0, 0, 1,
	}}

	rz := Mat4x4{[16]float32{
		float32(math.Cos(float64(az))), float32(math.Sin(float64(az))), 0, 0,
		-float32(math.Sin(float64(az))), float32(math.Cos(float64(az))), 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}}

	ryRx := rz.Mult(&ry)
	return ryRx.Mult(&rx)
}

func (a *Mat4x4) Mult(b *Mat4x4) Mat4x4 {
	var r Mat4x4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				r.M[i+j*4] += a.M[i+k*4] * b.M[k+j*4]
			}
		}
	}
	return r
}

func mat4x4Lookat(cameraPos, target, up Vec3) Mat4x4 {
	forward := target.Sub(cameraPos).Normalize()
	right := forward.Cross(up).Normalize()
	realUp := right.Cross(forward)

	return Mat4x4{[16]float32{
		right.X, realUp.X, -forward.X, 0,
		right.Y, realUp.Y, -forward.Y, 0,
		right.Z, realUp.Z, -forward.Z, 0,
		-right.Dot(cameraPos),
		-realUp.Dot(cameraPos),
		forward.Dot(cameraPos),
		1,
	}}
}

func mat4x4Perspective(fovY, aspect, znear, zfar float32) Mat4x4 {
	f := 1 / float32(math.Tan(float64(fovY)/2))
	return Mat4x4{[16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, -(zfar + znear) / (zfar - znear), -1,
		0, 0, -2 * znear * zfar / (zfar - znear), 0,
	}}
}

func (point Vec3) ToCamera(view, world *Mat4x4) Vec3 {
	vw := view.Mult(world)
	return point.Transform(&vw)
}

func (point Vec3) RotateAround(pivot Vec3, rotation *Mat4x4) Vec3 {
	rel := point.Sub(pivot)
	rotated := rel.Transform(rotation)
	return rotated.Add(pivot)
}

func (v Vec3) Project(fov float32) Vec2 {
	s := fov / (fov + v.Z)
	return Vec2{v.X * s, v.Y * s, s}
}

func planeCreate(normal Vec3, distance float32) Plane {
	return Plane{Normal: normal, Distance: distance}
}

func planeFromFace(faceVerts []Vec3) Plane {
	edge1 := faceVerts[1].Sub(faceVerts[0])
	edge2 := faceVerts[2].Sub(faceVerts[0])
	normal := edge1.Cross(edge2).Normalize()
	distance := -normal.Dot(faceVerts[0])

	boundary := make([]Vec3, len(faceVerts))
	copy(boundary, faceVerts)

	return Plane{
		Normal:        normal,
		Distance:      distance,
		Boundary:      boundary,
		BoundaryCount: len(faceVerts),
	}
}

func (p *Plane) IntersectLine(p1, p2 Vec3, out *Vec3) bool {
	dir := p2.Sub(p1)
	denom := p.Normal.Dot(dir)

	if math.Abs(float64(denom)) < 1e-10 {
		return false
	}

	t := -(p.Normal.Dot(p1) + p.Distance) / denom

	if t < 0 || t > 1 {
		return false
	}

	hit := p1.Add(dir.Scale(t))

	if p.Boundary != nil && !p.ContainsPoint(hit) {
		return false
	}

	*out = hit
	return true
}

func (p *Plane) ContainsPoint(pnt Vec3) bool {
	for i := 0; i < len(p.Boundary); i++ {
		a := p.Boundary[i]
		b := p.Boundary[(i+1)%len(p.Boundary)]
		edge := b.Sub(a)
		toPoint := pnt.Sub(a)
		if edge.Cross(toPoint).Dot(p.Normal) < 0 {
			return false
		}
	}
	return true
}

type colorState struct {
	R, G, B, A float32
}

type drawState struct {
	Fill   colorState
	Stroke colorState
	LineW  float32
	Effect EffectMode
	Grad2  colorState
}

const maxStack = 64

// Renderer kapselt den gesamten OpenGL-/Fensterzustand.
type Renderer struct {
	window    *glfw.Window
	program   uint32
	screenW   float32
	screenH   float32
	startTime float64

	// Vollbild-Umschaltung (F11): Start im Fenstermodus,
	// Größe/Position merken für die Rückkehr zum Fenstermodus.
	isFullscreen       bool
	winPrevW, winPrevH int
	winPosX, winPosY   int

	mouseX      float32
	mouseY      float32
	mouseStatus int

	locPos        int32
	locModelView  int32
	locProjection int32
	locPointSize  int32
	locMode       int32
	locColor      int32
	locColor2     int32
	locTime       int32
	locCenter     int32
	locRadius     int32
	locFogNear    int32
	locFogFar     int32
	locFogColor   int32

	stateStack    [maxStack]drawState
	stateStackTop int
	state         drawState
}

var renderer = &Renderer{}

func parseColorHex(hex string) colorState {
	c := colorState{1, 1, 1, 1}
	h := hex
	if len(h) > 0 && h[0] == '#' {
		h = h[1:]
	}

	if len(h) == 3 {
		buf := string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		n, err := strconv.ParseUint(buf, 16, 32)
		if err != nil {
			return c
		}
		c.R = float32((n>>16)&0xff) / 255
		c.G = float32((n>>8)&0xff) / 255
		c.B = float32(n&0xff) / 255
		return c
	}

	if len(h) == 6 {
		n, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return c
		}
		c.R = float32((n>>16)&0xff) / 255
		c.G = float32((n>>8)&0xff) / 255
		c.B = float32(n&0xff) / 255
		return c
	}

	return c
}

func parseColorRGB(r, g, b float32) colorState {
	return colorState{r / 255, g / 255, b / 255, 1}
}

func parseColorRGBA(r, g, b, a float32) colorState {
	return colorState{r / 255, g / 255, b / 255, a / 255}
}

func (r *Renderer) compileShader(shaderType uint32, src string) uint32 {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var ok int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		log := make([]byte, 1024)
		var length int32
		gl.GetShaderInfoLog(shader, 1024, &length, &log[0])
		fmt.Fprintf(os.Stderr, "Shader-Fehler: %s\n", string(log[:length]))
	}
	return shader
}

func (r *Renderer) createProgram(vertSrc, fragSrc string) uint32 {
	prog := gl.CreateProgram()
	vs := r.compileShader(gl.VERTEX_SHADER, vertSrc)
	fs := r.compileShader(gl.FRAGMENT_SHADER, fragSrc)
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)

	var ok int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		log := make([]byte, 1024)
		var length int32
		gl.GetProgramInfoLog(prog, 1024, &length, &log[0])
		fmt.Fprintf(os.Stderr, "Programm-Fehler: %s\n", string(log[:length]))
	}

	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	return prog
}

func (m *Mat4x4) Flatten() [16]float32 {
	var dst [16]float32
	for i := 0; i < 16; i++ {
		dst[i] = m.M[i]
	}
	return dst
}

func (r *Renderer) applyUniforms(useStroke bool) {
	col := r.state.Fill
	if useStroke {
		col = r.state.Stroke
	}
	mode := int32(r.state.Effect)

	gl.Uniform1i(r.locMode, mode)
	gl.Uniform4f(r.locColor, col.R, col.G, col.B, col.A)
	gl.Uniform4f(r.locColor2, r.state.Grad2.R, r.state.Grad2.G, r.state.Grad2.B, r.state.Grad2.A)
}

func (r *Renderer) drawVertices(verts []float32, mode uint32) {
	var buf uint32
	gl.GenBuffers(1, &buf)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(uint32(r.locPos))
	gl.VertexAttribPointer(uint32(r.locPos), 3, gl.FLOAT, false, 0, nil)
	gl.DrawArrays(mode, 0, int32(len(verts)/3))
	gl.DeleteBuffers(1, &buf)
}

func shapeMetrics3d(pts []float32) (cx, cy, cz, r float32) {
	n := len(pts) / 3
	for i := 0; i < len(pts); i += 3 {
		cx += pts[i]
		cy += pts[i+1]
		cz += pts[i+2]
	}
	cx /= float32(n)
	cy /= float32(n)
	cz /= float32(n)

	for i := 0; i < len(pts); i += 3 {
		dx := pts[i] - cx
		dy := pts[i+1] - cy
		dz := pts[i+2] - cz
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if dist > r {
			r = dist
		}
	}
	return
}

func (r *Renderer) cursorCallback(w *glfw.Window, xpos, ypos float64) {
	r.mouseX = float32(xpos) - r.screenW/2
	r.mouseY = -(float32(ypos) - r.screenH/2)
}

func (r *Renderer) mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button == glfw.MouseButtonLeft {
		if action == glfw.Press {
			r.mouseStatus = 1
		}
		if action == glfw.Release {
			r.mouseStatus = 2
		}
	}
}

func (r *Renderer) keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press {
		return
	}

	if key == glfw.KeyEscape {
		w.SetShouldClose(true)
	}
	if key == glfw.KeyF11 {
		r.ToggleFullscreen()
	}
}

// framebufferSizeCallback wird bei jeder Größenänderung des Framebuffers
// aufgerufen (z. B. beim Umschalten Vollbild <-> Fenster oder beim Ziehen).
func (r *Renderer) framebufferSizeCallback(w *glfw.Window, width, height int) {
	r.screenW = float32(width)
	r.screenH = float32(height)
	gl.Viewport(0, 0, int32(width), int32(height))
}

func (r *Renderer) Init(w, h int) bool {
	if err := glfw.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "GLFW-Init fehlgeschlagen.")
		return false
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	// Wir starten im Fenstermodus mit der gewünschten Größe w x h.
	// (Vollbild ist auf manchen Treibern/Wayland-Konfigurationen
	// problematisch beim Erstellen des GL-Kontexts.) Über F11 kann
	// später via ToggleFullscreen() in den Vollbildmodus gewechselt
	// werden.
	var err error
	r.window, err = glfw.CreateWindow(w, h, "lib3d_opengl_go", nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fenster-Erstellung fehlgeschlagen.")
		glfw.Terminate()
		return false
	}

	r.window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "GL-Init fehlgeschlagen.")
		return false
	}
	r.window.SetCursorPosCallback(r.cursorCallback)
	r.window.SetMouseButtonCallback(r.mouseButtonCallback)
	r.window.SetFramebufferSizeCallback(r.framebufferSizeCallback)
	r.window.SetKeyCallback(r.keyCallback)

	fbW, fbH := r.window.GetFramebufferSize()
	r.framebufferSizeCallback(r.window, fbW, fbH)

	// Fenstergröße für die Rückkehr aus dem Vollbildmodus merken.
	r.isFullscreen = false
	r.winPrevW, r.winPrevH = w, h

	vertSrc, err := os.ReadFile("shaders/vert.glsl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Shader-Dateien nicht gefunden.")
		return false
	}
	fragSrc, err := os.ReadFile("shaders/frag.glsl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Shader-Dateien nicht gefunden.")
		return false
	}

	r.program = r.createProgram(string(vertSrc), string(fragSrc))
	gl.UseProgram(r.program)

	r.locPos = gl.GetAttribLocation(r.program, gl.Str("aPos\x00"))
	r.locModelView = gl.GetUniformLocation(r.program, gl.Str("uModelView\x00"))
	r.locProjection = gl.GetUniformLocation(r.program, gl.Str("uProjection\x00"))
	r.locPointSize = gl.GetUniformLocation(r.program, gl.Str("uPointSize\x00"))
	r.locMode = gl.GetUniformLocation(r.program, gl.Str("uMode\x00"))
	r.locColor = gl.GetUniformLocation(r.program, gl.Str("uColor\x00"))
	r.locColor2 = gl.GetUniformLocation(r.program, gl.Str("uColor2\x00"))
	r.locTime = gl.GetUniformLocation(r.program, gl.Str("uTime\x00"))
	r.locCenter = gl.GetUniformLocation(r.program, gl.Str("uShapeCenter\x00"))
	r.locRadius = gl.GetUniformLocation(r.program, gl.Str("uShapeRadius\x00"))
	r.locFogNear = gl.GetUniformLocation(r.program, gl.Str("uFogNear\x00"))
	r.locFogFar = gl.GetUniformLocation(r.program, gl.Str("uFogFar\x00"))
	r.locFogColor = gl.GetUniformLocation(r.program, gl.Str("uFogColor\x00"))

	gl.Uniform1f(r.locPointSize, 4.0)
	gl.Uniform3f(r.locCenter, 0, 0, 0)
	gl.Uniform1f(r.locRadius, 1)

	identity := [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	gl.UniformMatrix4fv(r.locProjection, 1, false, &identity[0])
	gl.UniformMatrix4fv(r.locModelView, 1, false, &identity[0])

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.PROGRAM_POINT_SIZE)

	gl.Viewport(0, 0, int32(r.screenW), int32(r.screenH))
	r.startTime = glfw.GetTime()
	r.stateStackTop = -1

	r.state.Fill = colorState{1, 1, 1, 1}
	r.state.Stroke = colorState{0, 0, 0, 1}
	r.state.LineW = 1
	r.state.Effect = EffectFlat
	r.state.Grad2 = colorState{0, 0, 0, 1}

	return true
}

// ToggleFullscreen schaltet zwischen Fenster- und Vollbildmodus um
// (Demo: Taste F11).
func (r *Renderer) ToggleFullscreen() {
	if r.window == nil {
		return
	}

	if !r.isFullscreen {
		monitor := glfw.GetPrimaryMonitor()
		if monitor == nil {
			return
		}
		mode := monitor.GetVideoMode()
		if mode == nil {
			return
		}

		// Aktuelle Fenstergröße/-position merken, damit zurückgeschaltet
		// werden kann.
		r.winPosX, r.winPosY = r.window.GetPos()
		r.winPrevW, r.winPrevH = r.window.GetSize()

		r.window.SetMonitor(monitor, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
		r.isFullscreen = true
	} else {
		r.window.SetMonitor(nil, r.winPosX, r.winPosY, r.winPrevW, r.winPrevH, 0)
		r.isFullscreen = false
	}
}

func (r *Renderer) ShouldClose() bool {
	return r.window.ShouldClose()
}

func (r *Renderer) BeginFrame() {
	t := glfw.GetTime() - r.startTime
	gl.Uniform1f(r.locTime, float32(t))
}

func (r *Renderer) EndFrame() {
	r.window.SwapBuffers()
	glfw.PollEvents()
}

// StartAnimation startet die Render-Schleife. draw wird in jedem Frame
// aufgerufen. Kehrt zurück, wenn das Fenster geschlossen wird (ESC/F11
// werden vom Renderer behandelt). Die Schleife kapselt BeginFrame/EndFrame,
// sodass der Aufrufer nur noch das Zeichnen (und ggf. die Simulation)
// bereitstellen muss.
func (r *Renderer) StartAnimation(draw func()) {
	for !r.window.ShouldClose() {
		r.BeginFrame()
		if draw != nil {
			draw()
		}
		r.EndFrame()
	}
	r.Close()
}

// Close beendet GLFW. Beim Beenden des GL-Kontexts werden alle GL-Objekte
// (Meshes, Shader, VAOs) automatisch freigegeben.
func (r *Renderer) Close() {
	glfw.Terminate()
}

func (r *Renderer) SetFog(near, far, red, green, blue, alpha float32) {
	gl.Uniform1f(r.locFogNear, near)
	gl.Uniform1f(r.locFogFar, far)
	gl.Uniform4f(r.locFogColor, red, green, blue, alpha)
}

func (r *Renderer) Width() int      { return int(r.screenW) }
func (r *Renderer) Height() int     { return int(r.screenH) }
func (r *Renderer) Aspect() float32 { return r.screenW / r.screenH }
func (r *Renderer) MouseX() float32 { return r.mouseX }
func (r *Renderer) MouseY() float32 { return r.mouseY }
func (r *Renderer) Time() float32   { return float32(glfw.GetTime() - r.startTime) }

func (r *Renderer) IsMouseDown() bool { return r.mouseStatus == 1 }

func (r *Renderer) IsMouseUp() bool {
	if r.mouseStatus == 2 {
		r.mouseStatus = 0
		return true
	}
	return false
}

func (r *Renderer) SetProjection(m *Mat4x4) {
	fm := m.Flatten()
	gl.UniformMatrix4fv(r.locProjection, 1, false, &fm[0])
}

func (r *Renderer) SetModelview(m *Mat4x4) {
	fm := m.Flatten()
	gl.UniformMatrix4fv(r.locModelView, 1, false, &fm[0])
}

func (r *Renderer) SetGradientCenter(cx, cy, cz, radius float32) {
	gl.Uniform3f(r.locCenter, cx, cy, cz)
	gl.Uniform1f(r.locRadius, radius)
}

func (r *Renderer) Push() {
	if r.stateStackTop < maxStack-1 {
		r.stateStackTop++
		r.stateStack[r.stateStackTop] = r.state
	}
}

func (r *Renderer) Pop() {
	if r.stateStackTop >= 0 {
		r.state = r.stateStack[r.stateStackTop]
		r.stateStackTop--
	}
}

func (r *Renderer) SetFillColor(red, green, blue, alpha float32) {
	r.state.Fill = parseColorRGBA(red, green, blue, alpha)
}
func (r *Renderer) SetFillColorHex(hex string) { r.state.Fill = parseColorHex(hex) }
func (r *Renderer) SetStrokeColor(red, green, blue, alpha float32) {
	r.state.Stroke = parseColorRGBA(red, green, blue, alpha)
}
func (r *Renderer) SetStrokeColorHex(hex string) { r.state.Stroke = parseColorHex(hex) }
func (r *Renderer) SetStrokeWidth(width float32) { r.state.LineW = width }
func (r *Renderer) SetEffect(mode EffectMode)    { r.state.Effect = mode }
func (r *Renderer) SetGradient(red, green, blue, alpha float32) {
	r.state.Grad2 = parseColorRGBA(red, green, blue, alpha)
}
func (r *Renderer) SetGradientHex(hex string) { r.state.Grad2 = parseColorHex(hex) }

func (r *Renderer) Background(red, green, blue float32) {
	c := parseColorRGB(red, green, blue)
	gl.ClearColor(c.R, c.G, c.B, c.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (r *Renderer) BackgroundHex(hex string) {
	c := parseColorHex(hex)
	gl.ClearColor(c.R, c.G, c.B, c.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (r *Renderer) SetPointSize(px float32) {
	gl.Uniform1f(r.locPointSize, px)
}

func (r *Renderer) Point(x, y, z float32) {
	r.applyUniforms(true)
	v := []float32{x, y, z}
	r.drawVertices(v, gl.POINTS)
}

func (r *Renderer) Line(x1, y1, z1, x2, y2, z2 float32) {
	r.applyUniforms(true)
	v := []float32{x1, y1, z1, x2, y2, z2}
	r.drawVertices(v, gl.LINES)
}

func (r *Renderer) Triangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float32, style DrawStyle) {
	pts := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3}
	shapeMetrics3d(pts)

	if style == Fill || style == Both {
		r.applyUniforms(false)
		r.drawVertices(pts, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		r.applyUniforms(true)
		s := []float32{x1, y1, z1, x2, y2, z2, x2, y2, z2, x3, y3, z3, x3, y3, z3, x1, y1, z1}
		r.drawVertices(s, gl.LINES)
	}
}

func (r *Renderer) Shape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float32, style DrawStyle) {
	pts := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4}
	shapeMetrics3d(pts)

	if style == Fill || style == Both {
		r.applyUniforms(false)
		f := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3, x1, y1, z1, x3, y3, z3, x4, y4, z4}
		r.drawVertices(f, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		r.applyUniforms(true)
		s := []float32{x1, y1, z1, x2, y2, z2, x2, y2, z2, x3, y3, z3, x3, y3, z3, x4, y4, z4, x4, y4, z4, x1, y1, z1}
		r.drawVertices(s, gl.LINES)
	}
}

func (r *Renderer) Rect(x, y, w, h float32, style DrawStyle, z float32) {
	hw := w / 2
	hh := h / 2
	r.Shape(x-hw, y-hh, z, x+hw, y-hh, z, x+hw, y+hh, z, x-hw, y+hh, z, style)
}

func (r *Renderer) Circle(x, y, z, radius float32, style DrawStyle, segments int) {
	tau := float32(2 * math.Pi)

	if style == Fill || style == Both {
		fill := make([]float32, segments*9)
		fi := 0
		for i := 0; i < segments; i++ {
			a0 := (float32(i) / float32(segments)) * tau
			a1 := (float32(i+1) / float32(segments)) * tau
			fill[fi] = x
			fill[fi+1] = y
			fill[fi+2] = z
			fi += 3
			fill[fi] = x + float32(math.Cos(float64(a0)))*radius
			fill[fi+1] = y + float32(math.Sin(float64(a0)))*radius
			fill[fi+2] = z
			fi += 3
			fill[fi] = x + float32(math.Cos(float64(a1)))*radius
			fill[fi+1] = y + float32(math.Sin(float64(a1)))*radius
			fill[fi+2] = z
			fi += 3
		}
		r.applyUniforms(false)
		r.drawVertices(fill, gl.TRIANGLES)
	}

	if style == Stroke || style == Both {
		stroke := make([]float32, segments*3)
		si := 0
		for i := 0; i < segments; i++ {
			a := (float32(i) / float32(segments)) * tau
			stroke[si] = x + float32(math.Cos(float64(a)))*radius
			stroke[si+1] = y + float32(math.Sin(float64(a)))*radius
			stroke[si+2] = z
			si += 3
		}
		r.applyUniforms(true)
		r.drawVertices(stroke, gl.LINE_LOOP)
	}
}

func (r *Renderer) Polygon(pts []float32, style DrawStyle) {
	if len(pts) < 4 {
		return
	}
	shapeMetrics3d(pts)

	if style == Fill || style == Both {
		tris := len(pts)/3 - 2
		fill := make([]float32, tris*9)
		fi := 0
		for i := 1; i < len(pts)/3-1; i++ {
			fill[fi] = pts[0]
			fill[fi+1] = pts[1]
			fill[fi+2] = pts[2]
			fi += 3
			fill[fi] = pts[i*3]
			fill[fi+1] = pts[i*3+1]
			fill[fi+2] = pts[i*3+2]
			fi += 3
			fill[fi] = pts[i*3+3]
			fill[fi+1] = pts[i*3+4]
			fill[fi+2] = pts[i*3+5]
			fi += 3
		}
		r.applyUniforms(false)
		r.drawVertices(fill, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		r.applyUniforms(true)
		r.drawVertices(pts, gl.LINE_LOOP)
	}
}

func (r *Renderer) CreateMesh(flatVerts []float32) uint32 {
	var buf uint32
	gl.GenBuffers(1, &buf)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, len(flatVerts)*4, gl.Ptr(flatVerts), gl.STATIC_DRAW)
	return buf
}

func (r *Renderer) DrawMesh(buf uint32, vertCount int) {
	r.applyUniforms(true)
	gl.LineWidth(r.state.LineW)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.EnableVertexAttribArray(uint32(r.locPos))
	gl.VertexAttribPointer(uint32(r.locPos), 3, gl.FLOAT, false, 0, nil)
	gl.DrawArrays(gl.LINES, 0, int32(vertCount))
}

func (r *Renderer) DeleteMesh(buf uint32) {
	gl.DeleteBuffers(1, &buf)
}

func solidCreate() *Solid {
	return &Solid{}
}

func (s *Solid) Init(vertices []Vec3, edges []int) {
	s.Vertices = make([]Vec3, len(vertices))
	copy(s.Vertices, vertices)
	s.VertexCount = len(vertices)

	s.Edges = make([]int, len(edges))
	copy(s.Edges, edges)
	s.EdgeCount = len(edges) / 2

	s.MeshBuffer = 0
	s.MeshVertexCount = 0
}

func (s *Solid) ensureMesh() {
	if s.MeshBuffer != 0 {
		return
	}

	totalFloats := s.EdgeCount * 2 * 3
	verts := make([]float32, totalFloats)
	idx := 0
	for i := 0; i < s.EdgeCount; i++ {
		a := s.Vertices[s.Edges[i*2]]
		b := s.Vertices[s.Edges[i*2+1]]
		verts[idx] = a.X
		verts[idx+1] = a.Y
		verts[idx+2] = a.Z
		idx += 3
		verts[idx] = b.X
		verts[idx+1] = b.Y
		verts[idx+2] = b.Z
		idx += 3
	}

	s.MeshBuffer = renderer.CreateMesh(verts)
	s.MeshVertexCount = s.EdgeCount * 2
}

func (s *Solid) Draw(view, world *Mat4x4) {
	s.ensureMesh()
	vw := view.Mult(world)
	renderer.SetModelview(&vw)
	renderer.DrawMesh(s.MeshBuffer, s.MeshVertexCount)
}

func (s *Solid) Retain() {
	s.RefCount++
}

func (s *Solid) Release() {
	s.RefCount--
	if s.RefCount > 0 {
		return
	}
	if s.MeshBuffer != 0 {
		renderer.DeleteMesh(s.MeshBuffer)
	}
	s.Vertices = nil
	s.Edges = nil
	s.Faces = nil
	s.FaceSizes = nil
}

func solidBox(w, h, d float32) *Solid {
	hw := w / 2
	hh := h / 2
	hd := d / 2

	verts := []Vec3{
		{-hw, -hh, -hd}, {hw, -hh, -hd}, {hw, hh, -hd}, {-hw, hh, -hd},
		{-hw, -hh, hd}, {hw, -hh, hd}, {hw, hh, hd}, {-hw, hh, hd},
	}

	edgePairs := [12][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, {4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}
	edges := make([]int, 24)
	for i := 0; i < 12; i++ {
		edges[i*2] = edgePairs[i][0]
		edges[i*2+1] = edgePairs[i][1]
	}

	rawFaces := []int{
		0, 3, 2, 1, 4, 5, 6, 7, 0, 4, 7, 3,
		1, 2, 6, 5, 0, 1, 5, 4, 3, 7, 6, 2,
	}

	s := solidCreate()
	s.Init(verts, edges)

	s.FaceCount = 6
	s.Faces = make([]int, 24)
	copy(s.Faces, rawFaces)
	s.FaceSizes = make([]int, 6)
	for i := 0; i < 6; i++ {
		s.FaceSizes[i] = 4
	}

	return s
}

func solidPyramid(base, height float32) *Solid {
	hb := base / 2
	verts := []Vec3{
		{-hb, -height / 2, -hb}, {hb, -height / 2, -hb}, {hb, -height / 2, hb},
		{-hb, -height / 2, hb}, {0, height / 2, 0},
	}
	edgePairs := [8][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, {0, 4}, {1, 4}, {2, 4}, {3, 4},
	}
	edges := make([]int, 16)
	for i := 0; i < 8; i++ {
		edges[i*2] = edgePairs[i][0]
		edges[i*2+1] = edgePairs[i][1]
	}

	rawFaces := []int{
		0, 1, 4, 1, 2, 4, 2, 3, 4, 3, 0, 4, 3, 2, 1, 0,
	}
	sizes := []int{3, 3, 3, 3, 4}

	s := solidCreate()
	s.Init(verts, edges)

	s.FaceCount = 5
	s.Faces = make([]int, len(rawFaces))
	copy(s.Faces, rawFaces)
	s.FaceSizes = make([]int, 5)
	copy(s.FaceSizes, sizes)

	return s
}

func solidGrid(size float32, cells int) *Solid {
	half := size / 2
	step := size / float32(cells)
	stride := cells + 1

	verts := make([]Vec3, stride*stride)
	vi := 0
	for iz := 0; iz <= cells; iz++ {
		for ix := 0; ix <= cells; ix++ {
			verts[vi] = vec3(-half+float32(ix)*step, 0, -half+float32(iz)*step)
			vi++
		}
	}

	maxEdges := cells*stride*2 + stride*cells*2
	edges := make([]int, maxEdges*2)
	ei := 0

	for iz := 0; iz <= cells; iz++ {
		for ix := 0; ix < cells; ix++ {
			idx := iz*stride + ix
			edges[ei] = idx
			edges[ei+1] = idx + 1
			ei += 2
		}
	}
	for ix := 0; ix <= cells; ix++ {
		for iz := 0; iz < cells; iz++ {
			idx := iz*stride + ix
			edges[ei] = idx
			edges[ei+1] = idx + stride
			ei += 2
		}
	}

	s := solidCreate()
	s.Init(verts, edges[:ei])
	return s
}

func solidSphere(radius float32, slices, stacks int) *Solid {
	stride := slices + 1
	vcount := (stacks + 1) * stride
	verts := make([]Vec3, vcount)
	vi := 0

	for i := 0; i <= stacks; i++ {
		theta := (float32(i) / float32(stacks)) * float32(math.Pi)
		y := radius * float32(math.Cos(float64(theta)))
		r := radius * float32(math.Sin(float64(theta)))
		for j := 0; j <= slices; j++ {
			phi := (float32(j) / float32(slices)) * 2 * float32(math.Pi)
			verts[vi] = vec3(r*float32(math.Cos(float64(phi))), y, r*float32(math.Sin(float64(phi))))
			vi++
		}
	}

	maxEdges := (slices+1)*stacks + (stacks+1)*slices
	edges := make([]int, maxEdges*2)
	ei := 0

	for j := 0; j <= slices; j++ {
		for i := 0; i < stacks; i++ {
			a := i*stride + j
			b := (i+1)*stride + j
			edges[ei] = a
			edges[ei+1] = b
			ei += 2
		}
	}
	for i := 0; i <= stacks; i++ {
		for j := 0; j < slices; j++ {
			a := i*stride + j
			b := i*stride + j + 1
			edges[ei] = a
			edges[ei+1] = b
			ei += 2
		}
	}

	s := solidCreate()
	s.Init(verts, edges[:ei])
	return s
}
