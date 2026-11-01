package pygame

import "math"

type glMatrix [16]float64 // OpenGL column-major storage.
type glVector [4]float64

func glIdentity() glMatrix { return glMatrix{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }
func glMultiply(a, b glMatrix) (out glMatrix) {
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			for k := 0; k < 4; k++ {
				out[col*4+row] += a[k*4+row] * b[col*4+k]
			}
		}
	}
	return
}
func (m glMatrix) apply(v glVector) (out glVector) {
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			out[row] += m[col*4+row] * v[col]
		}
	}
	return
}
func glTranslation(x, y, z float64) glMatrix {
	m := glIdentity()
	m[12], m[13], m[14] = x, y, z
	return m
}
func glRotation(angle, x, y, z float64) glMatrix {
	length := math.Sqrt(x*x + y*y + z*z)
	if length == 0 {
		return glIdentity()
	}
	x, y, z = x/length, y/length, z/length
	s, c := math.Sincos(angle * math.Pi / 180)
	t := 1 - c
	return glMatrix{
		t*x*x + c, t*x*y + s*z, t*x*z - s*y, 0,
		t*x*y - s*z, t*y*y + c, t*y*z + s*x, 0,
		t*x*z + s*y, t*y*z - s*x, t*z*z + c, 0,
		0, 0, 0, 1,
	}
}
func glPerspective(fovy, aspect, near, far float64) glMatrix {
	f := 1 / math.Tan(fovy*math.Pi/360)
	return glMatrix{f / aspect, 0, 0, 0, 0, f, 0, 0, 0, 0, (far + near) / (near - far), -1, 0, 0, 2 * near * far / (near - far), 0}
}

type glVertex struct{ clip, color glVector }

func interpolateGL(a, b glVertex, t float64) (out glVertex) {
	for i := 0; i < 4; i++ {
		out.clip[i] = a.clip[i] + (b.clip[i]-a.clip[i])*t
		out.color[i] = a.color[i] + (b.color[i]-a.color[i])*t
	}
	return
}
func glPlane(v glVertex, plane int) float64 {
	axis := plane / 2
	if plane%2 == 0 {
		return v.clip[3] + v.clip[axis]
	}
	return v.clip[3] - v.clip[axis]
}

// Clip in homogeneous space before dividing by w. This also handles geometry
// crossing the near plane or behind the camera without screen-space explosions.
func clipGLPolygon(vertices []glVertex) []glVertex {
	for plane := 0; plane < 6 && len(vertices) > 0; plane++ {
		out := make([]glVertex, 0, len(vertices)+1)
		previous := vertices[len(vertices)-1]
		pd := glPlane(previous, plane)
		for _, current := range vertices {
			cd := glPlane(current, plane)
			if (cd >= 0) != (pd >= 0) {
				out = append(out, interpolateGL(previous, current, pd/(pd-cd)))
			}
			if cd >= 0 {
				out = append(out, current)
			}
			previous, pd = current, cd
		}
		vertices = out
	}
	return vertices
}
func clipGLLine(a, b glVertex) (glVertex, glVertex, bool) {
	for plane := 0; plane < 6; plane++ {
		da, db := glPlane(a, plane), glPlane(b, plane)
		if da < 0 && db < 0 {
			return a, b, false
		}
		if (da < 0) != (db < 0) {
			intersection := interpolateGL(a, b, da/(da-db))
			if da < 0 {
				a = intersection
			} else {
				b = intersection
			}
		}
	}
	return a, b, true
}
