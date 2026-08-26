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

func vec3Add(a, b Vec3) Vec3 {
	return vec3(a.X+b.X, a.Y+b.Y, a.Z+b.Z)
}

func vec3Sub(a, b Vec3) Vec3 {
	return vec3(a.X-b.X, a.Y-b.Y, a.Z-b.Z)
}

func vec3Scale(a Vec3, s float32) Vec3 {
	return vec3(a.X*s, a.Y*s, a.Z*s)
}

func vec3Mag(a Vec3, m float32) Vec3 {
	l := vec3Length(a)
	if l == 0 {
		return vec3(0, 0, 0)
	}
	return vec3Scale(a, m/l)
}

func vec3Limit(a Vec3, max float32) Vec3 {
	mSq := vec3SquaredLength(a)
	if mSq > max*max {
		return vec3Scale(a, max/float32(math.Sqrt(float64(mSq))))
	}
	return a
}

func vec3Negate(a Vec3) Vec3 {
	return vec3(-a.X, -a.Y, -a.Z)
}

func vec3Dot(a, b Vec3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func vec3Cross(a, b Vec3) Vec3 {
	return vec3(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func vec3SquaredLength(a Vec3) float32 {
	return a.X*a.X + a.Y*a.Y + a.Z*a.Z
}

func vec3Length(a Vec3) float32 {
	return float32(math.Sqrt(float64(vec3SquaredLength(a))))
}

func vec3Distance(a, b Vec3) float32 {
	return vec3Length(vec3Sub(a, b))
}

func vec3Normalize(a Vec3) Vec3 {
	l := vec3Length(a)
	if l == 0 {
		return vec3(0, 0, 0)
	}
	return vec3Scale(a, 1/l)
}

func vec3Lerp(a, b Vec3, t float32) Vec3 {
	return vec3Add(a, vec3Scale(vec3Sub(b, a), t))
}

func vec3Clone(a Vec3) Vec3 {
	return a
}

func vec3Equals(a, b Vec3) bool {
	return math.Abs(float64(a.X-b.X)) < epsilon &&
		math.Abs(float64(a.Y-b.Y)) < epsilon &&
		math.Abs(float64(a.Z-b.Z)) < epsilon
}

func vec3Transform(v Vec3, m *Mat4x4) Vec3 {
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

	ryRx := mat4x4Mult(&rz, &ry)
	return mat4x4Mult(&ryRx, &rx)
}

func mat4x4Mult(a, b *Mat4x4) Mat4x4 {
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
	forward := vec3Normalize(vec3Sub(target, cameraPos))
	right := vec3Normalize(vec3Cross(forward, up))
	realUp := vec3Cross(right, forward)

	return Mat4x4{[16]float32{
		right.X, realUp.X, -forward.X, 0,
		right.Y, realUp.Y, -forward.Y, 0,
		right.Z, realUp.Z, -forward.Z, 0,
		-vec3Dot(right, cameraPos),
		-vec3Dot(realUp, cameraPos),
		vec3Dot(forward, cameraPos),
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

func worldToCamera(point Vec3, view, world *Mat4x4) Vec3 {
	vw := mat4x4Mult(view, world)
	return vec3Transform(point, &vw)
}

func rotateAround(point, pivot Vec3, rotation *Mat4x4) Vec3 {
	rel := vec3Sub(point, pivot)
	rotated := vec3Transform(rel, rotation)
	return vec3Add(rotated, pivot)
}

func project(fov float32, v Vec3) Vec2 {
	s := fov / (fov + v.Z)
	return Vec2{v.X * s, v.Y * s, s}
}

func planeCreate(normal Vec3, distance float32) Plane {
	return Plane{Normal: normal, Distance: distance}
}

func planeFromFace(faceVerts []Vec3) Plane {
	edge1 := vec3Sub(faceVerts[1], faceVerts[0])
	edge2 := vec3Sub(faceVerts[2], faceVerts[0])
	normal := vec3Normalize(vec3Cross(edge1, edge2))
	distance := -vec3Dot(normal, faceVerts[0])

	boundary := make([]Vec3, len(faceVerts))
	copy(boundary, faceVerts)

	return Plane{
		Normal:        normal,
		Distance:      distance,
		Boundary:      boundary,
		BoundaryCount: len(faceVerts),
	}
}

func planeIntersectLine(p *Plane, p1, p2 Vec3, out *Vec3) bool {
	dir := vec3Sub(p2, p1)
	denom := vec3Dot(p.Normal, dir)

	if math.Abs(float64(denom)) < 1e-10 {
		return false
	}

	t := -(vec3Dot(p.Normal, p1) + p.Distance) / denom

	if t < 0 || t > 1 {
		return false
	}

	hit := vec3Add(p1, vec3Scale(dir, t))

	if p.Boundary != nil && !isPointInConvexPolygon(hit, p.Boundary, p.Normal) {
		return false
	}

	*out = hit
	return true
}

func isPointInConvexPolygon(p Vec3, polygon []Vec3, normal Vec3) bool {
	for i := 0; i < len(polygon); i++ {
		a := polygon[i]
		b := polygon[(i+1)%len(polygon)]
		edge := vec3Sub(b, a)
		toPoint := vec3Sub(p, a)
		if vec3Dot(vec3Cross(edge, toPoint), normal) < 0 {
			return false
		}
	}
	return true
}

var (
	window    *glfw.Window
	gMonitor  *glfw.Monitor
	program   uint32
	screenW   float32
	screenH   float32
	startTime float64

	gMouseX      float32
	gMouseY      float32
	gMouseStatus int

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
)

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

var (
	stateStack    [maxStack]drawState
	stateStackTop = -1
	state         drawState
)

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

func compileShader(shaderType uint32, src string) uint32 {
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

func createProgram(vertSrc, fragSrc string) uint32 {
	prog := gl.CreateProgram()
	vs := compileShader(gl.VERTEX_SHADER, vertSrc)
	fs := compileShader(gl.FRAGMENT_SHADER, fragSrc)
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

func flattenMatrix(src *Mat4x4) [16]float32 {
	var dst [16]float32
	for i := 0; i < 16; i++ {
		dst[i] = src.M[i]
	}
	return dst
}

func applyUniforms(useStroke bool) {
	col := state.Fill
	if useStroke {
		col = state.Stroke
	}
	mode := int32(state.Effect)

	gl.Uniform1i(locMode, mode)
	gl.Uniform4f(locColor, col.R, col.G, col.B, col.A)
	gl.Uniform4f(locColor2, state.Grad2.R, state.Grad2.G, state.Grad2.B, state.Grad2.A)
}

func drawVertices3d(verts []float32, mode uint32) {
	var buf uint32
	gl.GenBuffers(1, &buf)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(uint32(locPos))
	gl.VertexAttribPointer(uint32(locPos), 3, gl.FLOAT, false, 0, nil)
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

func cursorCallback(w *glfw.Window, xpos, ypos float64) {
	gMouseX = float32(xpos) - screenW/2
	gMouseY = -(float32(ypos) - screenH/2)
}

func mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button == glfw.MouseButtonLeft {
		if action == glfw.Press {
			gMouseStatus = 1
		}
		if action == glfw.Release {
			gMouseStatus = 2
		}
	}
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press {
		return
	}

	if key == glfw.KeyEscape {
		w.SetShouldClose(true)
	}
	// F11: Umschalten in den Fenstermodus deaktiviert
}

func renderInit(w, h int) bool {
	if err := glfw.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "GLFW-Init fehlgeschlagen.")
		return false
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	gMonitor = glfw.GetPrimaryMonitor()
	mode := gMonitor.GetVideoMode()
	if mode == nil {
		fmt.Fprintln(os.Stderr, "Video-Modus nicht verfügbar.")
		glfw.Terminate()
		return false
	}

	var err error
	window, err = glfw.CreateWindow(mode.Width, mode.Height, "lib3d_opengl_go", gMonitor, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fenster-Erstellung fehlgeschlagen.")
		glfw.Terminate()
		return false
	}

	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "GL-Init fehlgeschlagen.")
		return false
	}
	window.SetCursorPosCallback(cursorCallback)
	window.SetMouseButtonCallback(mouseButtonCallback)
	window.SetKeyCallback(keyCallback)

	fbW, fbH := window.GetFramebufferSize()
	screenW = float32(fbW)
	screenH = float32(fbH)
	w = fbW
	h = fbH

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

	program = createProgram(string(vertSrc), string(fragSrc))
	gl.UseProgram(program)

	locPos = gl.GetAttribLocation(program, gl.Str("aPos\x00"))
	locModelView = gl.GetUniformLocation(program, gl.Str("uModelView\x00"))
	locProjection = gl.GetUniformLocation(program, gl.Str("uProjection\x00"))
	locPointSize = gl.GetUniformLocation(program, gl.Str("uPointSize\x00"))
	locMode = gl.GetUniformLocation(program, gl.Str("uMode\x00"))
	locColor = gl.GetUniformLocation(program, gl.Str("uColor\x00"))
	locColor2 = gl.GetUniformLocation(program, gl.Str("uColor2\x00"))
	locTime = gl.GetUniformLocation(program, gl.Str("uTime\x00"))
	locCenter = gl.GetUniformLocation(program, gl.Str("uShapeCenter\x00"))
	locRadius = gl.GetUniformLocation(program, gl.Str("uShapeRadius\x00"))
	locFogNear = gl.GetUniformLocation(program, gl.Str("uFogNear\x00"))
	locFogFar = gl.GetUniformLocation(program, gl.Str("uFogFar\x00"))
	locFogColor = gl.GetUniformLocation(program, gl.Str("uFogColor\x00"))

	gl.Uniform1f(locPointSize, 4.0)
	gl.Uniform3f(locCenter, 0, 0, 0)
	gl.Uniform1f(locRadius, 1)

	identity := [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	gl.UniformMatrix4fv(locProjection, 1, false, &identity[0])
	gl.UniformMatrix4fv(locModelView, 1, false, &identity[0])

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.PROGRAM_POINT_SIZE)

	gl.Viewport(0, 0, int32(w), int32(h))
	startTime = glfw.GetTime()

	state.Fill = colorState{1, 1, 1, 1}
	state.Stroke = colorState{0, 0, 0, 1}
	state.LineW = 1
	state.Effect = EffectFlat
	state.Grad2 = colorState{0, 0, 0, 1}

	return true
}

func renderShouldClose() bool {
	return window.ShouldClose()
}

func renderFrameBegin() {
	t := glfw.GetTime() - startTime
	gl.Uniform1f(locTime, float32(t))
}

func renderFrameEnd() {
	window.SwapBuffers()
	glfw.PollEvents()
}

func renderSetFog(near, far, r, g, b, a float32) {
	gl.Uniform1f(locFogNear, near)
	gl.Uniform1f(locFogFar, far)
	gl.Uniform4f(locFogColor, r, g, b, a)
}

func renderGetWidth() int      { return int(screenW) }
func renderGetHeight() int     { return int(screenH) }
func renderGetAspect() float32 { return screenW / screenH }
func renderMouseX() float32    { return gMouseX }
func renderMouseY() float32    { return gMouseY }
func renderGetTime() float32   { return float32(glfw.GetTime() - startTime) }

func renderIsMouseDown() bool { return gMouseStatus == 1 }

func renderIsMouseUp() bool {
	if gMouseStatus == 2 {
		gMouseStatus = 0
		return true
	}
	return false
}

func renderSetProjection(m *Mat4x4) {
	fm := flattenMatrix(m)
	gl.UniformMatrix4fv(locProjection, 1, false, &fm[0])
}

func renderSetModelview(m *Mat4x4) {
	fm := flattenMatrix(m)
	gl.UniformMatrix4fv(locModelView, 1, false, &fm[0])
}

func renderSetGradientCenter(cx, cy, cz, radius float32) {
	gl.Uniform3f(locCenter, cx, cy, cz)
	gl.Uniform1f(locRadius, radius)
}

func renderPush() {
	if stateStackTop < maxStack-1 {
		stateStackTop++
		stateStack[stateStackTop] = state
	}
}

func renderPop() {
	if stateStackTop >= 0 {
		state = stateStack[stateStackTop]
		stateStackTop--
	}
}

func renderFillColor(r, g, b, a float32)   { state.Fill = parseColorRGBA(r, g, b, a) }
func renderFillColorHex(hex string)        { state.Fill = parseColorHex(hex) }
func renderStrokeColor(r, g, b, a float32) { state.Stroke = parseColorRGBA(r, g, b, a) }
func renderStrokeColorHex(hex string)      { state.Stroke = parseColorHex(hex) }
func renderStrokeWidth(w float32)          { state.LineW = w }
func renderSetEffect(mode EffectMode)      { state.Effect = mode }
func renderSetGradient(r, g, b, a float32) { state.Grad2 = parseColorRGBA(r, g, b, a) }
func renderSetGradientHex(hex string)      { state.Grad2 = parseColorHex(hex) }

func renderBackground(r, g, b float32) {
	c := parseColorRGB(r, g, b)
	gl.ClearColor(c.R, c.G, c.B, c.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func renderBackgroundHex(hex string) {
	c := parseColorHex(hex)
	gl.ClearColor(c.R, c.G, c.B, c.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func renderPointSize(px float32) {
	gl.Uniform1f(locPointSize, px)
}

func renderPoint(x, y, z float32) {
	applyUniforms(true)
	v := []float32{x, y, z}
	drawVertices3d(v, gl.POINTS)
}

func renderLine(x1, y1, z1, x2, y2, z2 float32) {
	applyUniforms(true)
	v := []float32{x1, y1, z1, x2, y2, z2}
	drawVertices3d(v, gl.LINES)
}

func renderTriangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float32, style DrawStyle) {
	pts := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3}
	shapeMetrics3d(pts)

	if style == Fill || style == Both {
		applyUniforms(false)
		drawVertices3d(pts, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		applyUniforms(true)
		s := []float32{x1, y1, z1, x2, y2, z2, x2, y2, z2, x3, y3, z3, x3, y3, z3, x1, y1, z1}
		drawVertices3d(s, gl.LINES)
	}
}

func renderShape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float32, style DrawStyle) {
	pts := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4}
	shapeMetrics3d(pts)

	if style == Fill || style == Both {
		applyUniforms(false)
		f := []float32{x1, y1, z1, x2, y2, z2, x3, y3, z3, x1, y1, z1, x3, y3, z3, x4, y4, z4}
		drawVertices3d(f, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		applyUniforms(true)
		s := []float32{x1, y1, z1, x2, y2, z2, x2, y2, z2, x3, y3, z3, x3, y3, z3, x4, y4, z4, x4, y4, z4, x1, y1, z1}
		drawVertices3d(s, gl.LINES)
	}
}

func renderRect(x, y, w, h float32, style DrawStyle, z float32) {
	hw := w / 2
	hh := h / 2
	renderShape(x-hw, y-hh, z, x+hw, y-hh, z, x+hw, y+hh, z, x-hw, y+hh, z, style)
}

func renderCircle(x, y, z, radius float32, style DrawStyle, segments int) {
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
		applyUniforms(false)
		drawVertices3d(fill, gl.TRIANGLES)
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
		applyUniforms(true)
		drawVertices3d(stroke, gl.LINE_LOOP)
	}
}

func renderPolygon(pts []float32, style DrawStyle) {
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
		applyUniforms(false)
		drawVertices3d(fill, gl.TRIANGLES)
	}
	if style == Stroke || style == Both {
		applyUniforms(true)
		drawVertices3d(pts, gl.LINE_LOOP)
	}
}

func renderCreateMesh(flatVerts []float32) uint32 {
	var buf uint32
	gl.GenBuffers(1, &buf)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, len(flatVerts)*4, gl.Ptr(flatVerts), gl.STATIC_DRAW)
	return buf
}

func renderDrawMesh(buf uint32, vertCount int) {
	applyUniforms(true)
	gl.LineWidth(state.LineW)
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.EnableVertexAttribArray(uint32(locPos))
	gl.VertexAttribPointer(uint32(locPos), 3, gl.FLOAT, false, 0, nil)
	gl.DrawArrays(gl.LINES, 0, int32(vertCount))
}

func renderDeleteMesh(buf uint32) {
	gl.DeleteBuffers(1, &buf)
}

func solidCreate() *Solid {
	return &Solid{}
}

func solidInit(s *Solid, vertices []Vec3, edges []int) {
	s.Vertices = make([]Vec3, len(vertices))
	copy(s.Vertices, vertices)
	s.VertexCount = len(vertices)

	s.Edges = make([]int, len(edges))
	copy(s.Edges, edges)
	s.EdgeCount = len(edges) / 2

	s.MeshBuffer = 0
	s.MeshVertexCount = 0
}

func ensureMesh(s *Solid) {
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

	s.MeshBuffer = renderCreateMesh(verts)
	s.MeshVertexCount = s.EdgeCount * 2
}

func solidDraw(s *Solid, view, world *Mat4x4) {
	ensureMesh(s)
	vw := mat4x4Mult(view, world)
	renderSetModelview(&vw)
	renderDrawMesh(s.MeshBuffer, s.MeshVertexCount)
}

func solidRetain(s *Solid) {
	s.RefCount++
}

func solidRelease(s *Solid) {
	s.RefCount--
	if s.RefCount > 0 {
		return
	}
	if s.MeshBuffer != 0 {
		renderDeleteMesh(s.MeshBuffer)
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
	solidInit(s, verts, edges)

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
	solidInit(s, verts, edges)

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
	solidInit(s, verts, edges[:ei])
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
	solidInit(s, verts, edges[:ei])
	return s
}
