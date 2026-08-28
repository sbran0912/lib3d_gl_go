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

func NewBody(solid *Solid, x, y, z float32, cfg BodyConfig) Body {
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
	return b
}

func (b *Body) GetFacePlanes(worldVerts []Vec3) PlaneArray {
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

func (b *Body) Draw(view *Mat4x4) {
	t := mat4x4Translate(b.Pos.X, b.Pos.Y, b.Pos.Z)
	var world Mat4x4
	if b.RotX == 0 && b.RotY == 0 && b.RotZ == 0 {
		world = t
	} else {
		rot := mat4x4Rotate(b.RotX, b.RotY, b.RotZ)
		world = t.Mult(&rot)
	}

	renderer.SetStrokeWidth(b.LineWidth)
	renderer.SetStrokeColorHex(b.Color)
	renderer.SetFillColorHex(b.Color) // Füllfarbe für die Flächen

	b.Solid.Draw(view, &world)
}

func (a *Body) Distance(b *Body) float32 {
	return a.Pos.Distance(b.Pos)
}
