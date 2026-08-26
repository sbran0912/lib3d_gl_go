package main

import (
	"math"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Vehicle ist ein autonomes Fahrzeug der Simulation.
type Vehicle struct {
	Body    Body
	Vel     Vec3
	Accel   Vec3
	Heading Vec3
	Health  float32
	DNA     [4]float32
}

// vehicCreate erstellt ein neues Vehicle mit zufälliger DNA.
func vehicCreate(body Body) Vehicle {
	return Vehicle{
		Body:    body,
		Vel:     vec3(0, 0, 0),
		Accel:   vec3(0, 0, 0),
		Heading: vec3(0, 0, 0),
		Health:  1,
		DNA: [4]float32{
			randomFloat(-1.0, 1.0),  // Force to Poison
			randomFloat(-1.0, 1.0),  // Force to good Food
			randomFloat(20.0, 60.0), // Radius to Poison
			randomFloat(20.0, 60.0), // Radius to good Food
		},
	}
}

// vehicDestroy gibt die Ressourcen des Fahrzeugs frei.
func vehicDestroy(v *Vehicle) {
	bodyDestroy(&v.Body)
}

// vehicAlignToVelocity richtet den Body an der Geschwindigkeit aus.
func vehicAlignToVelocity(v *Vehicle) {
	vel := v.Vel

	mag := float32(math.Sqrt(float64(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)))
	if mag < 0.0001 {
		return
	}

	magXZ := float32(math.Sqrt(float64(vel.X*vel.X + vel.Z*vel.Z)))

	// vel.Y/mag muss in [-1,1] liegen (Schutz vor NaN durch Float-Rundung)
	cosY := vel.Y / mag
	if cosY > 1 {
		cosY = 1
	} else if cosY < -1 {
		cosY = -1
	}
	v.Body.RotX = float32(math.Acos(float64(cosY)))
	if magXZ < 0.0001 {
		v.Body.RotY = 0
	} else {
		v.Body.RotY = float32(math.Atan2(float64(vel.X/magXZ), float64(vel.Z/magXZ)))
	}
	v.Body.RotZ = 0

	// heading als normalisierte Richtung ableiten
	v.Heading = vec3(vel.X/mag, vel.Y/mag, vel.Z/mag)
}

// vehicApplyForce addiert eine Kraft zur Beschleunigung.
func vehicApplyForce(v *Vehicle, force Vec3) {
	v.Accel = vec3Add(v.Accel, force)
}

// vehicUpdate integriert die Bewegung und begrenzt die Geschwindigkeit.
func vehicUpdate(v *Vehicle) {
	v.Vel = vec3Add(v.Vel, v.Accel)
	speed := vec3Length(v.Vel)
	v.Vel = vec3Scale(vec3Normalize(v.Vel), constrainNum(speed, 0.5, 2.0))

	v.Accel = vec3(0, 0, 0)
	v.Body.Pos = vec3Add(v.Body.Pos, v.Vel)
}

// vehicSeek steuert das Fahrzeug in Richtung eines Ziels.
func vehicSeek(v *Vehicle, target Vec3, isBadfood bool) {
	desired := vec3Limit(vec3Sub(target, v.Body.Pos), 3.0)
	if isBadfood {
		desired = vec3Scale(desired, v.DNA[0])
	} else {
		desired = vec3Scale(desired, v.DNA[1])
	}

	steer := vec3Limit(vec3Sub(desired, v.Vel), 2.0)
	vehicApplyForce(v, vec3Scale(steer, 0.2))
}

// foodCreate erzeugt count Food-Bodies an zufälligen Positionen.
func foodCreate(count int, color string, mesh *Solid) []Body {
	food := make([]Body, 0, count)
	for range count {
		singleFood := bodyCreate(mesh,
			float32(random(-100, 100)),
			float32(random(-100, 100)),
			float32(random(-100, 100)),
			BodyConfig{Color: color, LineWidth: 1.0})
		food = append(food, singleFood)
	}
	return food
}

// foodRespawn ergänzt Food, falls weniger als min vorhanden sind.
func foodRespawn(food []Body, mesh *Solid, min, count int, color string) []Body {
	if len(food) < min {
		for range count {
			singleFood := bodyCreate(mesh,
				float32(random(-100, 100)),
				float32(random(-100, 100)),
				float32(random(-100, 100)),
				BodyConfig{Color: color, LineWidth: 1.0})
			food = append(food, singleFood)
		}
	}
	return food
}

// vehicleEatFood sucht das nächste (gute/schlechte) Food im DNA-Radius
// und isst es auf, sobald es nahe genug ist.
func vehicleEatFood(v *Vehicle, food *[]Body, isBadfood bool) {
	minDist := float32(math.Inf(1))
	idx := -1

	filter := v.DNA[3]
	if isBadfood {
		filter = v.DNA[2]
	}

	for i := range *food {
		d := vec3Distance(v.Body.Pos, (*food)[i].Pos)
		if d < filter && d < minDist {
			minDist = d
			idx = i
		}
	}

	if idx > -1 {
		if vec3Distance(v.Body.Pos, (*food)[idx].Pos) < 3 {
			bodyDestroy(&(*food)[idx]) // Food aufessen (Refcount freigeben)
			*food = append((*food)[:idx], (*food)[idx+1:]...)
			if isBadfood {
				v.Health -= 0.1
			} else {
				v.Health += 0.1
			}
		} else {
			vehicSeek(v, (*food)[idx].Pos, isBadfood)
		}
	}
}

// vehicBoundary reflektiert die Geschwindigkeit an den Weltgrenzen.
func vehicBoundary(v *Vehicle) {
	// Definiere deine Weltgrenzen
	const (
		minX = -130.0
		maxX = 130.0
		minY = -130.0
		maxY = 130.0
		minZ = -130.0
		maxZ = 130.0
	)

	// X-Achse
	if v.Body.Pos.X < minX || v.Body.Pos.X > maxX {
		v.Vel.X *= -1.0
	}

	// Y-Achse (Höhe)
	if v.Body.Pos.Y < minY || v.Body.Pos.Y > maxY {
		v.Vel.Y *= -1.0
	}

	// Z-Achse (Tiefe)
	if v.Body.Pos.Z < minZ || v.Body.Pos.Z > maxZ {
		v.Vel.Z *= -1.0
	}
}

// vehicIsDead meldet, ob die Gesundheit des Fahrzeugs erschöpft ist.
func vehicIsDead(v *Vehicle) bool {
	return v.Health < 0.0
}

func main() {
	runtime.LockOSThread()

	if !renderInit(1600, 1000) {
		return
	}
	camPos := vec3(50, 100, 200)
	target := vec3(0, 0, 0)
	up := vec3(0, 1, 0)
	renderSetFog(100.0, 400.0, 0.25, 0.25, 0.25, 1.0)

	randomInit()

	gridMesh := solidGrid(600, 24)
	grid := bodyCreate(gridMesh, 0, 0, 0, BodyConfig{Color: "#777774", LineWidth: 1.0})

	foodMesh := solidSphere(3, 8, 8)
	poison := foodCreate(30, "#FF0000", foodMesh)
	food := foodCreate(30, "#44ff44", foodMesh)

	vehicMesh := solidPyramid(2, 6)
	vehics := make([]Vehicle, 0, 10)

	for i := 0; i < 10; i++ {
		vehic := vehicCreate(bodyCreate(vehicMesh, 0, 20, 100, bodyConfigDefault))
		vehic.Vel = vec3(randomFloat(-2, 2), randomFloat(-2, 2), randomFloat(-2, 2))
		vehics = append(vehics, vehic)
	}

	for !renderShouldClose() {
		renderFrameBegin()
		renderBackground(40, 40, 40)

		view := mat4x4Lookat(camPos, target, up)
		proj := mat4x4Perspective(1.2, renderGetAspect(), 0.1, 1000.0)
		renderSetProjection(&proj)

		bodyDraw(&grid, &view)

		food = foodRespawn(food, foodMesh, 20, 30, "#44ff44")
		poison = foodRespawn(poison, foodMesh, 20, 30, "#FF0000")

		getOlder := randomFloat(0, 1) < 0.015

		for i := len(vehics) - 1; i >= 0; i-- {
			v := &vehics[i]

			vehicBoundary(v)
			vehicleEatFood(v, &food, false)
			vehicleEatFood(v, &poison, true)
			vehicAlignToVelocity(v)
			vehicUpdate(v)

			if v.Health < 0.5 {
				v.Body.Color = "#FF0000"
			} else {
				v.Body.Color = "#ffffff"
			}

			bodyDraw(&v.Body, &view)

			if getOlder {
				v.Health -= 0.05
			}

			if vehicIsDead(v) {
				vehicDestroy(v)
				vehics = append(vehics[:i], vehics[i+1:]...)
			}
		}

		for i := range poison {
			bodyDraw(&poison[i], &view)
		}

		for i := range food {
			bodyDraw(&food[i], &view)
		}

		renderFrameEnd()
	}

	// Programm-Ende: alles zurücksetzen
	bodyDestroy(&grid)
	for i := range poison {
		bodyDestroy(&poison[i])
	}
	for i := range food {
		bodyDestroy(&food[i])
	}
	for i := range vehics {
		vehicDestroy(&vehics[i])
	}

	glfw.Terminate()
}
