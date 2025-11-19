package util

import "math"

// Mat3ToQuaternion converts a 3x3 rotation matrix to a quaternion.
// nolint:varnamelen
func Mat3ToQuaternion(m [3][3]float64) [4]float64 {
	var q [4]float64

	trace := m[0][0] + m[1][1] + m[2][2]

	switch {
	case trace > 0:
		s := math.Sqrt(trace+1.0) * 2 // S = 4 * qw
		q[3] = 0.25 * s
		q[0] = (m[2][1] - m[1][2]) / s
		q[1] = (m[0][2] - m[2][0]) / s
		q[2] = (m[1][0] - m[0][1]) / s
	case (m[0][0] > m[1][1]) && (m[0][0] > m[2][2]):
		s := math.Sqrt(1.0+m[0][0]-m[1][1]-m[2][2]) * 2 // S = 4 * qx
		q[3] = (m[2][1] - m[1][2]) / s
		q[0] = 0.25 * s
		q[1] = (m[0][1] + m[1][0]) / s
		q[2] = (m[0][2] + m[2][0]) / s
	case m[1][1] > m[2][2]:
		s := math.Sqrt(1.0+m[1][1]-m[0][0]-m[2][2]) * 2 // S = 4 * qy
		q[3] = (m[0][2] - m[2][0]) / s
		q[0] = (m[0][1] + m[1][0]) / s
		q[1] = 0.25 * s
		q[2] = (m[1][2] + m[2][1]) / s
	default:
		s := math.Sqrt(1.0+m[2][2]-m[0][0]-m[1][1]) * 2 // S = 4 * qz
		q[3] = (m[1][0] - m[0][1]) / s
		q[0] = (m[0][2] + m[2][0]) / s
		q[1] = (m[1][2] + m[2][1]) / s
		q[2] = 0.25 * s
	}

	return q
}

// QuaternionToMat3 converts a quaternion to a 3x3 rotation matrix.
// nolint:varnamelen
func QuaternionToMat3(q [4]float64) [3][3]float64 {
	x, y, z, w := q[0], q[1], q[2], q[3]
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z

	return [3][3]float64{
		{1 - 2*(yy+zz), 2 * (xy - wz), 2 * (xz + wy)},
		{2 * (xy + wz), 1 - 2*(xx+zz), 2 * (yz - wx)},
		{2 * (xz - wy), 2 * (yz + wx), 1 - 2*(xx+yy)},
	}
}
