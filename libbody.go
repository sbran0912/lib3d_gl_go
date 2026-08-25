package main

type Body struct {
	Solid     *Solid
	Pos       Vec3
	RotX      float32
	RotY      float32
	RotZ      float32
	Color     string
	LineWidth float32
}

type BodyConfig struct {
	Color     string
	LineWidth float32
	RotX      float32
	RotY      float32
	RotZ      float32
}

var bodyConfigDefault = BodyConfig{
	Color:     "#ffffff",
	LineWidth: 1.0,
}

type PlaneArray struct {
	Data  []Plane
	Count int
}

type Line struct {
	P1        Vec3
	P2        Vec3
	Color     string
	LineWidth float32
}

func bodyCreate(solid *Solid, x, y, z float32, cfg BodyConfig) Body {
	color := cfg.Color
	if color == "" {
		color = "#ffffff"
	}

	b := Body{
		Solid:     solid,
		Pos:       vec3(x, y, z),
		RotX:      cfg.RotX,
		RotY:      cfg.RotY,
		RotZ:      cfg.RotZ,
		Color:     color,
		LineWidth: cfg.LineWidth,
	}
	solidRetain(solid)
	return b
}

func bodyDestroy(b *Body) {
	solidRelease(b.Solid)
}

func bodyGetFacePlanes(b *Body, worldVerts []Vec3) PlaneArray {
	s := b.Solid
	if s.Faces == nil || s.FaceCount == 0 {
		return PlaneArray{}
	}

	planes := make([]Plane, s.FaceCount)
	fi := 0
	for i := 0; i < s.FaceCount; i++ {
		fsize := 4
		if s.FaceSizes != nil {
			fsize = s.FaceSizes[i]
		}
		faceVerts := make([]Vec3, fsize)
		for j := 0; j < fsize; j++ {
			vi := s.Faces[fi+j]
			faceVerts[j] = worldVerts[vi]
		}
		planes[i] = planeFromFace(faceVerts)
		fi += fsize
	}

	return PlaneArray{Data: planes, Count: s.FaceCount}
}

func planeArrayFree(pa *PlaneArray) {
	pa.Data = nil
	pa.Count = 0
}

func bodyDraw(b *Body, view *Mat4x4) {
	t := mat4x4Translate(b.Pos.X, b.Pos.Y, b.Pos.Z)
	var world Mat4x4
	if b.RotX == 0 && b.RotY == 0 && b.RotZ == 0 {
		world = t
	} else {
		rot := mat4x4Rotate(b.RotX, b.RotY, b.RotZ)
		world = mat4x4Mult(&t, &rot)
	}

	renderStrokeWidth(b.LineWidth)
	renderStrokeColorHex(b.Color)

	solidDraw(b.Solid, view, &world)
}

func bodyDistance(a, b *Body) float32 {
	return vec3Distance(a.Pos, b.Pos)
}

func lineCreate(p1, p2 Vec3, color string, width float32) *Line {
	return &Line{
		P1:        p1,
		P2:        p2,
		Color:     color,
		LineWidth: width,
	}
}

func lineDraw(l *Line, view *Mat4x4, toPoint *Vec3) {
	ident := mat4x4Identity()
	mv := mat4x4Mult(view, &ident)
	renderSetModelview(&mv)
	renderStrokeColorHex(l.Color)
	renderStrokeWidth(l.LineWidth)
	end := l.P2
	if toPoint != nil {
		end = *toPoint
	}
	renderLine(l.P1.X, l.P1.Y, l.P1.Z, end.X, end.Y, end.Z)
}
